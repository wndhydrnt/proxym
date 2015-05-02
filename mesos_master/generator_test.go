package mesos_master

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShouldGenerateOneService(t *testing.T) {
	domain := "mesos.example.com"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.RequestURI == "/master/state.json" {
			s := State{Leader: "master@10.10.10.10:5050"}

			data, err := json.Marshal(s)
			if err != nil {
				log.Fatal("Error marshalling apps")
			}

			w.Write(data)
			return
		}
	}))

	sg := MesosMasterServiceGenerator{
		config: &Config{
			Domain:       domain,
			Masters:      ts.URL,
			PollInterval: 1,
		},
		hc: &http.Client{},
	}

	services, _ := sg.Generate()

	require.Len(t, services, 1)

	require.Equal(t, services[0].Id, "/mesos-master")
	require.Equal(t, services[0].Domain, domain)
	require.Equal(t, services[0].Port, 80)
	require.Equal(t, services[0].Protocol, "tcp")
	require.Equal(t, services[0].Source, "Mesos Master")
	require.Equal(t, services[0].Hosts[0].Ip, "10.10.10.10")
	require.Equal(t, services[0].Hosts[0].Port, 5050)
}
