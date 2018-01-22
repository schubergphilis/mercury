package balancer

// Sticky Balance based on Stickiness, the provided ID should always match the same node
func Sticky(s []Statistics, id string) []Statistics {
	var matches []Statistics
	for _, stats := range s {
		if stats.UUID == id {
			matches = append(matches, stats)
			return matches
		}
	}
	// if no matches, return all nodes
	return s
}
