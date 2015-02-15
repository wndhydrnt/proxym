package mesos_master

import (
	"encoding/json"
	"errors"
	"github.com/kelseyhightower/envconfig"
	"github.com/wndhydrnt/proxym/log"
	"github.com/wndhydrnt/proxym/manager"
	"github.com/wndhydrnt/proxym/types"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Domain       string
	Enabled      bool
	Masters      string
	PollInterval int `env_config:"poll_interval"`
}

type State struct {
	Leader string
}

func parseLeader(leader string) (types.Host, error) {
	address := strings.Split(leader, "@")[1]

	parts := strings.Split(address, ":")

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return types.Host{}, err
	}

	return types.Host{Ip: parts[0], Port: port}, nil
}

func pickMaster(masters string) string {
	rand.Seed(time.Now().Unix())

	mastersList := strings.Split(masters, ",")

	return mastersList[rand.Intn(len(mastersList))]
}

func query(hc *http.Client, master string) (string, error) {
	var state State

	url := master + "/master/state.json"

	resp, err := hc.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(body, &state)
	if err != nil {
		return "", err
	}

	return state.Leader, nil
}

func sanitizeConfig(c *Config) error {
	if c.Domain == "" {
		return errors.New("'PROXYM_MESOS_MASTER_DOMAIN' not set")
	}

	if c.Masters == "" {
		return errors.New("'PROXYM_MESOS_MASTER_MASTERS' not set")
	}

	if c.PollInterval == 0 {
		c.PollInterval = 10
	}

	return nil
}

func init() {
	var c Config

	envconfig.Process("proxym_mesos_master", &c)

	if c.Enabled {
		err := sanitizeConfig(&c)
		if err != nil {
			log.ErrorLog.Error("Not initializing module Mesos Master: '%s'", err)
			return
		}

		hc := &http.Client{}

		n := &MesosMasterNotifier{
			config: &c,
			hc:     hc,
		}
		manager.AddNotifier(n)

		sg := &MesosMasterServiceGenerator{
			config: &c,
			hc:     hc,
		}
		manager.AddServiceGenerator(sg)
	}
}
