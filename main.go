package main

import (
	"code.google.com/p/go-html-transform/css/selector"
	"code.google.com/p/go-html-transform/h5"
	"code.google.com/p/go.net/html"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalln("./das_downloader email password")
	}

	email := os.Args[1]
	password := os.Args[2]

	cookieJar, _ := cookiejar.New(nil)
	client := http.Client{nil, nil, cookieJar}

	signIn(&client, email, password)

	screencastUrls := make(chan *url.URL, 5)

	numDownloadingRoutines := runtime.NumCPU()
	runtime.GOMAXPROCS(numDownloadingRoutines)
	wait := sync.WaitGroup{}

	for i := 0; i < numDownloadingRoutines; i++ {
		wait.Add(1)
		go func() {
			for screencastUrl := range screencastUrls {
				downloadScreencast(&client, screencastUrl)
			}
			wait.Done()
		}()
	}

	getScreencastUrls(&client, screencastUrls)
	close(screencastUrls)

	wait.Wait()
}

// grab https://www.destroyallsoftware.com/screencasts/users/sign_in
//  - store cookies
//  - get the form
//  - fill in user & pass
// submit the form (remember the cross site request hidden input!)
func signIn(client *http.Client, email, password string) {
	log.Println("Fetching DAS signin form")
	signInUrl := "https://www.destroyallsoftware.com/screencasts/users/sign_in"
	signInResponse, err := client.Get(signInUrl)
	if err != nil {
		log.Fatalf("Error getting signin form: %v\n", err)
	}

	matchingNodes := extractMatchingHtmlNodes(signInResponse, "form input")

	formParams := make(url.Values)

	for _, node := range matchingNodes {
		var name, value string
		for _, attr := range node.Attr {
			if attr.Key == "name" {
				name = attr.Val
			} else if attr.Key == "value" {
				value = attr.Val
			}
		}
		formParams.Set(name, value)
	}
	formParams.Set("user[email]", email)
	formParams.Set("user[password]", password)

	log.Println("Submitting Login Form")

	signinResponse, err := client.PostForm(signInUrl, formParams)
	if err != nil {
		log.Fatalf("Error signing in: %v", err)
	}
	rawBody, _ := ioutil.ReadAll(signinResponse.Body)
	body := string(rawBody)
	if strings.Contains(body, "Signed in successfully") {
		log.Println("Signed in OK")
	} else {
		log.Fatalln("Failed to login")
	}
}

// get list of all screencast pages from https://www.destroyallsoftware.com/screencasts/catalog
func getScreencastUrls(client *http.Client, screencastUrls chan *url.URL) {
	log.Println("Fetching Screencast Catalog")
	catalogResponse, err := client.Get("https://www.destroyallsoftware.com/screencasts/catalog")
	if err != nil {
		log.Fatalf("Error retreiving catalog page: %v\n", err)
	}

	// foreach screencast link (.screencast .title a)
	// TODO figure out why my real selector didn't work
	matchingNodes := extractMatchingHtmlNodes(catalogResponse, "a")

	log.Printf("Found %v matching screencast urls", len(matchingNodes))

	for _, node := range matchingNodes {
		for _, attr := range node.Attr {
			if attr.Key == "href" && strings.HasPrefix(attr.Val, "/screencasts/catalog/") {
				fullDownloadUrl := fmt.Sprintf("https://www.destroyallsoftware.com%v/download", attr.Val)
				url, err := url.Parse(fullDownloadUrl)
				if err == nil {
					screencastUrls <- url
				} else {
					log.Fatalf("Error parsing url %v with err: %v", attr.Val, err)
				}
			}
		}
	}
}

// Hit the screencast URL, stream the file into a local folder
// skip any files that already exist in the folder.
func downloadScreencast(client *http.Client, screencastUrl *url.URL) {
	log.Printf("Trying %v\n", screencastUrl)

	resp, err := client.Get(screencastUrl.String())
	if err != nil {
		log.Printf("ERROR downloading %v: %v", screencastUrl, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("%v for %v", resp.Status, screencastUrl)
	} else {

		split_file_path := strings.Split(resp.Request.URL.Path, "/")
		filename := split_file_path[len(split_file_path)-1]

		stat, err := os.Stat(filename)
		// os.Stat errors if the file doesn't exist, so check
		// presence of an error
		if err == nil {
			// get the downloaded file size & content length from the request
			existingFileSize := stat.Size()
			contentLength, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
			if err != nil {
				// The too hard basket
				log.Printf("File %v exists and failed to parse remote file size. Skipping (err %v)", err)
				return
			}

			if contentLength == existingFileSize {
				log.Printf("File %v is already fully downloaded. Skipping.", filename)
				return
			} else {
				log.Printf("File %v is partially downloaded. Retrying. (%v of %v)", filename, existingFileSize, contentLength)
			}
		}

		file, err := os.Create(filename)
		if err != nil {
			log.Printf("Error creating file %v: %v\n", filename, err)
			return
		}
		defer file.Close()

		log.Printf("Started writing %v", filename)
		n, err := io.Copy(file, resp.Body)
		log.Printf("Wrote %v bytes to %v\n\n", n, filename)
	}
}

func extractMatchingHtmlNodes(response *http.Response, cssSelector string) []*html.Node {
	tree, err := h5.New(response.Body)
	if err != nil {
		log.Fatalf("Error parsing body into tree: %v\n", err)
	}

	selectorChain, err := selector.Selector(cssSelector)
	if err != nil {
		log.Fatalf("Error parsing cssSelector %v: %v\n", cssSelector, err)
	}

	return selectorChain.Find(tree.Top())
}
