package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	// must provide a PORT
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}
	// twitter creds
	tc := twitterCreds{
		consumerKey:       os.Getenv("TWITTER_CONSUMER_KEY"),
		consumerSecret:    os.Getenv("TWITTER_CONSUMER_SECRET"),
		accessToken:       os.Getenv("TWITTER_ACCESS_TOKEN"),
		accessTokenSecret: os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"),
	}
	// hashtags to track
	track := []string{"#cat, #cats, #kitten, #kittie, #meow, #instacats, #instacat, #catsofinstagram, #catstagram, #cutecats, #kittycat"}

	// create a stop bool with an associated mutex
	// so we can access it from many goroutines at the same time
	var stoplock sync.Mutex
	stop := false
	// signalChan is sent any SIGINT or SIGTERM signals when
	// something tries to halt the program
	signalChan := make(chan os.Signal, 1)
	// stopChan is passed on to stream.start() as a way to
	// tell it to terminate its process when the program is stopped
	stopChan := make(chan struct{}, 1)
	// closeConn tells the request to the Twitter API to stop
	closeConn := make(chan struct{}, 1)

	go func() {
		// block until there is a signal on the signalChan
		<-signalChan
		// set stop to true when a signal is received
		stoplock.Lock()
		stop = true
		stoplock.Unlock()
		// stop stream and close the connection
		log.Println("Stopping...")
		stopChan <- struct{}{}
		closeConn <- struct{}{}
	}()
	// relays incoming SIGINT and SIGTERM signals to signalChan
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	// initialize the stream and start it
	stream := newStream(tc, track)
	twitterStoppedChan := stream.start(stopChan, closeConn)

	go func() {
		// infinite loop that closes the connection every minute
		// (a new connection will be established by the loop inside
		// stream.start()
		for {
			time.Sleep(5 * time.Minute)
			closeConn <- struct{}{}
			// check if we should break out of this loop
			stoplock.Lock()
			if stop {
				stoplock.Unlock()
				break
			}
			stoplock.Unlock()
		}
	}()

	// Setup handlers
	http.HandleFunc("/", index)
	http.Handle("/stream", stream)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}

	// block until the we have stopped reading the twitter stream
	<-twitterStoppedChan
}

func index(w http.ResponseWriter, r *http.Request) {
	err := template.Must(template.ParseFiles("index.html")).Execute(w, fmt.Sprintf("ws://%s/stream", r.Host))
	if err != nil {
		log.Print(err)
		http.Error(w, "error parsing index.html template", http.StatusInternalServerError)
	}
}
