package hipache

import (
	"github.com/stretchr/testify/assert"
	"github.com/wndhydrnt/proxym/types"
	"testing"
)

type driverMock struct {
	addedBackends    map[string][]string
	createdFrontends map[string]string
	listBackendsFunc func(string) map[string]struct{}
	removedBackends  map[string][]string
}

func (dm *driverMock) addBackend(key, backend string) error {
	_, ok := dm.addedBackends[key]
	if ok == false {
		dm.addedBackends[key] = []string{}
	}

	dm.addedBackends[key] = append(dm.addedBackends[key], backend)

	return nil
}

func (dm *driverMock) createFrontend(key, identifier string) error {
	dm.createdFrontends[key] = identifier

	return nil
}
func (dm *driverMock) listBackends(key string) (map[string]struct{}, error) {
	return dm.listBackendsFunc(key), nil
}
func (dm *driverMock) removeBackend(key, backend string) error {
	_, ok := dm.removedBackends[key]
	if ok == false {
		dm.removedBackends[key] = []string{}
	}

	dm.removedBackends[key] = append(dm.removedBackends[key], backend)

	return nil
}

func TestGenerate(t *testing.T) {
	mock := &driverMock{
		addedBackends:    make(map[string][]string),
		createdFrontends: make(map[string]string),
		listBackendsFunc: func(key string) map[string]struct{} {
			assert.Equal(t, "frontend:unit.test.devel", key)

			bs := make(map[string]struct{})
			bs["http://11.11.11.11:8888"] = struct{}{}

			return bs
		},
		removedBackends: make(map[string][]string),
	}

	services := []*types.Service{
		&types.Service{
			ApplicationProtocol: "http",
			Domains:             []string{"unit.test.devel"},
			Hosts:               []types.Host{types.Host{Ip: "10.10.10.10", Port: 8888}},
			Id:                  "unittest",
		},
	}

	hp := hipache{mock}

	hp.Generate(services)

	assert.Len(t, mock.addedBackends, 1)
	assert.Equal(t, "http://10.10.10.10:8888", mock.addedBackends["frontend:unit.test.devel"][0])

	assert.Len(t, mock.removedBackends, 1)
	assert.Equal(t, "http://11.11.11.11:8888", mock.removedBackends["frontend:unit.test.devel"][0])
}
