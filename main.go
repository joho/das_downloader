package main

import (
	"code.google.com/p/go-html-transform/css/selector"
	"code.google.com/p/go-html-transform/h5"
	"code.google.com/p/go.net/html"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalln("You must specify your DAAS login and password")
	}

	email := os.Args[1]
	password := os.Args[2]

	cookieJar, _ := cookiejar.New(nil)
	client := http.Client{nil, nil, cookieJar}

	signIn(&client, email, password)

	screencastUrls := make(chan *url.URL, 5)

	wait := sync.WaitGroup{}
	wait.Add(1)
	go func() {
		for screencastUrl := range screencastUrls {
			downloadScreencast(screencastUrl)
		}
		wait.Done()
	}()

	getScreencastUrls(&client, screencastUrls)
	close(screencastUrls)

	wait.Wait()
}

func signIn(client *http.Client, email, password string) {
	log.Println("Fetching DAS signin form")
	// grab https://www.destroyallsoftware.com/screencasts/users/sign_in
	//  - store cookies
	//  - get the form
	//  - fill in user & pass
	// submit the form (remember the cross site request hidden input!)
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

	_, err = client.PostForm(signInUrl, formParams)
	if err != nil {
		log.Fatalf("Error signing in: %v", err)
	}
	log.Println("Signed in to DAS")
}

func getScreencastUrls(client *http.Client, screencastUrls chan *url.URL) {
	log.Println("Fetching Screencast Catalog")
	// get list of all screencast pages from https://www.destroyallsoftware.com/screencasts/catalog
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
			if attr.Key == "href" && strings.HasPrefix(attr.Val, "/screencasts/catalog") {
				url, err := url.Parse(attr.Val)
				if err == nil {
					screencastUrls <- url
				} else {
					log.Fatalf("Error parsing url %v with err: %v", attr.Val, err)
				}
			}
		}
	}
}

func downloadScreencast(screencastUrl *url.URL) {
	//   - visit page
	//   - find the link with text "Download for Desktop"
	//   - follow it & anny redirect
	//   - save it to a folder
	log.Printf("TODO get from %v\n", screencastUrl)
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
