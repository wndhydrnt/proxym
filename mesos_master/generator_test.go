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

	require.Equal(t, "http", services[0].ApplicationProtocol)
	require.Equal(t, "/mesos-master", services[0].Id)
	require.Equal(t, domain, services[0].Domains[0])
	require.Equal(t, 80, services[0].Port)
	require.Equal(t, "tcp", services[0].TransportProtocol)
	require.Equal(t, "Mesos Master", services[0].Source)
	require.Equal(t, "10.10.10.10", services[0].Hosts[0].Ip)
	require.Equal(t, 5050, services[0].Hosts[0].Port)
}
