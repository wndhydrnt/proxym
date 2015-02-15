package marathon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/wndhydrnt/proxym/log"
	"io/ioutil"
	"net/http"
	"strings"
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

	server := strings.Split(wt.config.Servers, ",")[0]

	callbackUrl := fmt.Sprintf("http://%s:%s%s", wt.config.HttpHost, wt.config.HttpPort, wt.config.Endpoint)

	url := fmt.Sprintf("%s%s?callbackUrl=%s", server, eventubscriptionsEndpoint, callbackUrl)

	resp, err := wt.httpClient.Post(url, contentType, bytes.NewBufferString(""))
	if err != nil {
		log.ErrorLog.Error("Error registering callback with Marathon '%s' - disabling module 'Marathon'", err)
		return
	}

	if resp.StatusCode != 200 {
		log.ErrorLog.Error("Unable to register callback with Marathon server '%s' - disabling module 'Marathon'", server)
	}
}

func (wt *Watcher) callbackHandler(w http.ResponseWriter, r *http.Request) {
	var event Event

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.ErrorLog.Error("Error reading Marathon event '%s'", err)
		return
	}

	err = json.Unmarshal(body, &event)
	if err != nil {
		log.ErrorLog.Error("Error unmarshalling Marathon event '%s'", err)
		return
	}

	if event.EventType == "status_update_event" {
		select {
		case wt.refreshChannel <- "refresh":
			log.AppLog.Info("Triggering refresh")
		default:
		}
	}

	w.Write([]byte(""))
}
