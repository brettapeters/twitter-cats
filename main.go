package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
)

func main() {
	tc := twitterCreds{
		consumerKey:       os.Getenv("TWITTER_CONSUMER_KEY"),
		consumerSecret:    os.Getenv("TWITTER_CONSUMER_SECRET"),
		accessToken:       os.Getenv("TWITTER_ACCESS_TOKEN"),
		accessTokenSecret: os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"),
	}
	track := []string{"#cat, #cats, #kitten, #kittie, #meow, #instacats, #instacat, #catsofinstagram, #catstagram, #cutecats, #kittycat"}
	stream := newStream(tc, track)
	go stream.start()

	http.HandleFunc("/", index)
	http.Handle("/stream", stream)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	err := template.Must(template.ParseFiles("index.html")).Execute(w, nil)
	if err != nil {
		log.Fatal(err)
	}
}
