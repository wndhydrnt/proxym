package mesos_master

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestShouldTriggerRefreshWhenMasterChanges(t *testing.T) {
	refresh := make(chan string, 1)
	reqCount := 0
	wg := &sync.WaitGroup{}
	wg.Add(1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.RequestURI == "/master/state.json" {
			var s State

			if reqCount == 0 {
				s = State{Leader: "master@10.10.10.10:5050"}
			}

			if reqCount != 0 {
				s = State{Leader: "master@10.11.10.10:5050"}
			}

			reqCount = reqCount + 1

			data, err := json.Marshal(s)
			if err != nil {
				log.Fatal("Error marshalling apps")
			}

			w.Write(data)
			return
		}
	}))

	n := MesosMasterNotifier{
		config: &Config{
			Masters:      ts.URL,
			PollInterval: 1,
		},
		hc: &http.Client{},
	}

	go n.Start(refresh, make(chan int), wg)

	time.Sleep(3 * time.Second)

	select {
	case msg := <-refresh:
		if msg != "refresh" {
			require.FailNow(t, "Expect message from refresh channel to be of value 'refresh'")
		}
	default:
		require.FailNow(t, "Expect to receive message from refresh channel")
	}
}
