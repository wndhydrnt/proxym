package marathon

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShouldRegisterWithMarathon(t *testing.T) {
	callReceived := false
	refresh := make(chan string, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callReceived = true

		w.WriteHeader(http.StatusOK)

		fmt.Fprintln(w, "")
	}))
	defer ts.Close()

	c := &http.Client{}

	watcher := Watcher{
		config: &Config{
			HttpHost: "localhost",
			HttpPort: "56398",
			Endpoint: "/callback",
			Servers:  ts.URL,
		},
		httpClient: c,
	}

	watcher.Start(refresh)

	require.True(t, callReceived)
}

func TestReactsToStatusUpdateEvent(t *testing.T) {
	event := bytes.NewBufferString(`{"eventType": "status_update_event"}`)
	refresh := make(chan string, 1)

	req, err := http.NewRequest("POST", "http://localhost:9000/callback", event)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()

	watcher := Watcher{
		config:         &Config{},
		httpClient:     &http.Client{},
		refreshChannel: refresh,
	}

	watcher.callbackHandler(w, req)

	require.Equal(t, 200, w.Code)
	require.Equal(t, "", w.Body.String())

	select {
	case msg := <-refresh:
		if msg != "refresh" {
			require.FailNow(t, "Expect message from refresh channel to be of value 'refresh'")
		}
	default:
		require.FailNow(t, "Expect to receive message from refresh channel")
	}
}

func TestIgnoresDifferentEvent(t *testing.T) {
	event := bytes.NewBufferString(`{"eventType": "failed_health_check_event"}`)
	refresh := make(chan string, 1)

	req, err := http.NewRequest("POST", "http://localhost:9000/callback", event)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()

	watcher := Watcher{
		config:         &Config{},
		httpClient:     &http.Client{},
		refreshChannel: refresh,
	}

	watcher.callbackHandler(w, req)

	require.Equal(t, 200, w.Code)
	require.Equal(t, "", w.Body.String())

	select {
	case <-refresh:
		require.FailNow(t, "Expect no message to be received from refresh channel")
	default:
	}
}
