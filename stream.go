package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

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

func (s *stream) start() {
	params := map[string]string{
		"track": strings.Join(s.track, ","),
	}
	res, err := s.consumer.Post(resourceURL, params, s.token)
	if err != nil {
		log.Fatal(err, res)
	}
	go s.forwarder.run()
	tweetStream := json.NewDecoder(res.Body)
	for {
		var tweet Tweet
		err = tweetStream.Decode(&tweet)
		if err != nil {
			log.Fatal(err)
		}
		photoURL := tweet.GetPhotoURL()
		if photoURL != "" {
			s.forwarder.forward <- &tweet
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
		log.Fatal("ServeHTTP:", err)
	}
	client := newClient(socket)
	s.forwarder.join <- client
	defer func() {
		s.forwarder.leave <- client
	}()
	client.write()
}
