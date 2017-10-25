package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mrjones/oauth"
)

var resourceURL = "https://stream.twitter.com/1.1/statuses/filter.json"

type stream struct {
	consumer  *oauth.Consumer
	token     *oauth.AccessToken
	track     []string
	forwarder *forwarder
}

type twitterCreds struct {
	consumerKey       string
	consumerSecret    string
	accessToken       string
	accessTokenSecret string
}

func newStream(tc twitterCreds, track []string) *stream {
	return &stream{
		consumer: oauth.NewConsumer(tc.consumerKey, tc.consumerSecret, oauth.ServiceProvider{}),
		token: &oauth.AccessToken{
			Token:  tc.accessToken,
			Secret: tc.accessTokenSecret,
		},
		track:     track,
		forwarder: newForwarder(),
	}
}

func (s *stream) start(stopchan, closeConn <-chan struct{}) <-chan struct{} {
	// run the Tweet forwarder
	go s.forwarder.run()
	// stoppedchan is a signal channel that will be used
	// to communicate when the goroutine spawned here
	// is stopped
	stoppedchan := make(chan struct{}, 1)
	go func() {
		// when the goroutine exits, stoppedchan will
		// receive a signal
		defer func() {
			stoppedchan <- struct{}{}
		}()
		for {
			select {
			case <-stopchan:
				// stopchan is a receive-only channel that will tell
				// this goroutine to stop
				return
			default:
				// send a request to Twitter and read the stream
				log.Println("Reading tweets")
				s.readFromTwitter(closeConn)
				log.Println("Waiting to reconnect...")
				time.Sleep(10 * time.Second) // wait before reconnecting
			}
		}
	}()
	// return stoppedchan as a receive-only channel
	return stoppedchan
}

func (s *stream) readFromTwitter(closeConn <-chan struct{}) {
	// make list of params
	params := map[string]string{
		"track": strings.Join(s.track, ","),
	}
	// make the request
	res, err := s.consumer.Post(resourceURL, params, s.token)
	if err != nil {
		log.Println(err, res)
		return
	}
	defer res.Body.Close()
	// decode Tweets from the stream
	tweetStream := json.NewDecoder(res.Body)
	for {
		select {
		case <-closeConn:
			return
		default:
			var tweet Tweet
			err = tweetStream.Decode(&tweet)
			if err != nil {
				log.Println(err)
				return
			}
			// forward photo URLs to clients
			photoURL := tweet.GetPhotoURL()
			if photoURL != "" {
				s.forwarder.forward <- &tweet
			}
		}
	}
}

const (
	socketBufferSize  = 1024
	messageBufferSize = 256
)

var upgrader = &websocket.Upgrader{ReadBufferSize: socketBufferSize, WriteBufferSize: socketBufferSize}

func (s *stream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print(err)
		http.Error(w, "error upgrading protocols - "+err.Error(), http.StatusBadRequest)
	}
	client := newClient(socket)
	s.forwarder.join <- client
	defer func() {
		s.forwarder.leave <- client
	}()
	client.write()
}
