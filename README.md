# Destroy All Software Downloader

The all-you-can eat subscriptions to Destroy All Software finish up on May 31st. You might want to download them all locally before then.

I wrote this little tool to download them all for you in Go. I did it for the practice, rather than the time savings. I could have clicked all those links myself in about the same about of time.

## How?

I did a compile! You can download the [OSX Binary for DAS Downloader](http://johnbarton.s3.amazonaws.com/das_downloader) then it's as simple as

    ./das_downloader your_email_here your_password_here

And it should start downloading them all. It'll do it concurrently, doing as many downloads as you've got CPUs.

## Who?

Made by [John Barton](http://whoisjohnbarton.com), MIT licenced. If you want me to build you something, find out more at [codename.io](http://codename.io).
