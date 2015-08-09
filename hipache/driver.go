package hipache

import (
	"github.com/mediocregopher/radix.v2/redis"
)

type driver interface {
	addBackend(key, backend string) error
	createFrontend(key, identifier string) error
	listBackends(key string) (map[string]struct{}, error)
	removeBackend(key, backend string) error
}

type redisDriver struct {
	c *redis.Client
}

func (r *redisDriver) addBackend(key, backend string) error {
	return r.c.Cmd("RPUSH", key, backend).Err
}

func (r *redisDriver) createFrontend(key, identifier string) error {
	return r.c.Cmd("RPUSH", key, identifier).Err
}

func (r *redisDriver) listBackends(key string) (map[string]struct{}, error) {
	backends := make(map[string]struct{})

	storedBackends, err := r.c.Cmd("LRANGE", key, 0, -1).List()
	if err != nil {
		return backends, err
	}

	for index, storedBackend := range storedBackends {
		if index == 0 {
			continue
		}
		backends[storedBackend] = struct{}{}
	}

	return backends, nil
}

func (r *redisDriver) removeBackend(key, backend string) error {
	return r.c.Cmd("LREM", key, 0, backend).Err
}

func newRedisDriver(c *config) (*redisDriver, error) {
	r, err := redis.Dial("tcp", c.RedisAddress)
	if err != nil {
		return nil, err
	}
	return &redisDriver{c: r}, nil
}
