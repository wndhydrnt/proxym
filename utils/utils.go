package utils

import (
	"math/rand"
	"time"
)

func PickRandomFromList(list []string) string {
	if len(list) == 1 {
		return list[0]
	}

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	max := (len(list) * 100) - 1

	pick := float64(r.Intn(max)) / 100

	return list[int(pick)]
}
