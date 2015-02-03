package marathon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Receives messages via the HTTP event bus of Marathon.
type Watcher struct {
	config         *Config
	httpClient     *http.Client
	refreshChannel chan string
}

// Start a HTTP server and register its endpoint with Marathon in order to receive event notifications.
func (wt *Watcher) Start(refresh chan string) {
	wt.refreshChannel = refresh

	go func() {
		http.HandleFunc(wt.config.Endpoint, wt.callbackHandler)

		http.ListenAndServe(fmt.Sprintf("%s:%s", wt.config.HttpHost, wt.config.HttpPort), nil)
	}()

	server := wt.config.MarathonServers[0]

	callbackUrl := fmt.Sprintf("http://%s:%s%s", wt.config.HttpHost, wt.config.HttpPort, wt.config.Endpoint)

	url := fmt.Sprintf("%s%s?callbackUrl=%s", server, eventubscriptionsEndpoint, callbackUrl)

	resp, _ := wt.httpClient.Post(url, contentType, bytes.NewBufferString(""))

	if resp.StatusCode != 200 {
		log.Fatal(fmt.Sprintf("Unable to register callback with Marathon server '%s'", server))
	}
}

func (wt *Watcher) callbackHandler(w http.ResponseWriter, r *http.Request) {
	var event Event

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("marathon.Watcher.callbackHandler: Error reading Marathon event")
		return
	}

	err = json.Unmarshal(body, &event)
	if err != nil {
		log.Println("marathon.Watcher.callbackHandler: Error unmarshalling Marathon event")
		return
	}

	if event.EventType == "status_update_event" {
		select {
		case wt.refreshChannel <- "refresh":
		default:
		}
	}

	w.Write([]byte(""))
}
