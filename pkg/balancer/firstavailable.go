package balancer

// FirstAvailable Balance based on nothing, returns the first host entry
// this is used to limit the output to 1 host
func FirstAvailable(s []Statistics) []Statistics {
	var matches []Statistics
	if len(s) > 0 {
		matches = append(matches, s[0])
		return matches
	}
	// if no matches, return all nodes
	return s
}
