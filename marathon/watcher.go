package marathon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/wndhydrnt/proxym/log"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

// Receives messages via the HTTP event bus of Marathon.
type Watcher struct {
	config            *Config
	httpClient        *http.Client
	httpListenAddress string
	refreshChannel    chan string
}

// Start a HTTP server and register its endpoint with Marathon in order to receive event notifications.
func (wt *Watcher) Start(refresh chan string, quit chan int, wg *sync.WaitGroup) {
	defer wg.Done()

	wt.refreshChannel = refresh

	server := strings.Split(wt.config.Servers, ",")[0]

	callbackUrl := fmt.Sprintf("http://%s%s", wt.httpListenAddress, "/marathon/callback")

	url := fmt.Sprintf("%s%s?callbackUrl=%s", server, eventubscriptionsEndpoint, callbackUrl)

	resp, err := wt.httpClient.Post(url, contentType, bytes.NewBufferString(""))
	if err != nil {
		log.ErrorLog.Error("Error registering callback with Marathon '%s'", err)
		return
	}

	if resp.StatusCode != 200 {
		log.ErrorLog.Error("Unable to register callback with Marathon server '%s''", server)
	}
}

func (wt *Watcher) callbackHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
