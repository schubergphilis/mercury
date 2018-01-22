package balancer

import (
	"math/rand"
	"time"
)

// Random implements the Random balance type, by randomizing the array provided
func Random(s []Statistics) []Statistics {
	rand.Seed(time.Now().UTC().UnixNano())
	for i := len(s) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		s[i], s[j] = s[j], s[i]
	}
	return s
}
