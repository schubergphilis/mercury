package balancer

import (
	"math/rand"
	"sort"
	"time"
)

// Weighted based loadbalancing interface for statistic
type Weighted struct{ statistics }

// Less implements preference based loadbalancing by sorting based on Preference counter
func (s Weighted) Less(i, j int) bool {
	return s.statistics[i].Weighted > s.statistics[j].Weighted
}

/*
type numberSet struct {
	values []float64
	bounds []float64
}

func (s *numberSet) Len() int { return len(s.values) }
func (s *numberSet) Swap(i, j int) {
	s.values[i], s.values[j] = s.values[j], s.values[i]
	s.bounds[i], s.bounds[j] = s.bounds[j], s.bounds[i]
}
func (s *numberSet) Less(i, j int) bool { return s.bounds[i] < s.bounds[j] }
*/

// WeighCalculation implements a weighted form of loadbalancing
func WeighCalculation(s []Statistics) []Statistics {

	// the process:
	// order by weight (we keep all nodes in there, highest weight ends on top by default)
	// get sum of weight
	// generate a random number on sum of weight
	// match that based on the weights

	// order by weight (we fall back to the most weighted if anything happens)
	sort.Sort(Weighted{s})

	// first we take all weighted numbers, and make them 100%
	sum := int(0)
	for _, v := range s {
		//log.Printf("id: %d weight: %d uuid: %s", id, v.Weighted, v.UUID)
		sum += v.Weighted
	}
	//match := make(map[int]int)

	// no weights, return data as is
	if sum == 0 {
		return s
	}

	// pick a random number in the sum
	rand.Seed(time.Now().UTC().UnixNano())
	j := rand.Intn(sum)

	// found out which one matches
	mark := int(0)
	for i, v := range s {
		mark += v.Weighted
		if v.Weighted > 0 {
			if mark > j {
				// if the sum is greater then random, then this is our value to be on top
				s[i], s[0] = s[0], s[i]
				break
			}
		}
	}

	// go over the weighted stuff again
	//mark := 0
	//for i := len(s) - 1; i > 0; i-- {

	//}

	//if s[i].Weighted
	//mark += s[i].Weighted
	//}

	/*rand.Seed(time.Now().UTC().UnixNano())
	for i := len(s) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		s[i], s[j] = s[j], s[i]
	}
	return s*/
	return s

}
