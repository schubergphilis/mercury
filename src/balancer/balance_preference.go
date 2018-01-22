package balancer

// Preference based loadbalancing interface for statistic
type Preference struct{ statistics }

// Less implements preference based loadbalancing by sorting based on Preference counter
func (s Preference) Less(i, j int) bool {
	return s.statistics[i].Preference < s.statistics[j].Preference
}
