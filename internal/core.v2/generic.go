package core

func sliceAddedAndDeleted(old, new []string) (added []string, deleted []string) {
	return sliceAdded(old, new), sliceAdded(new, old)
}

// find added items
func sliceAdded(old, new []string) (added []string) {
	oldList := make(map[string]struct{})
	for _, i := range old {
		oldList[i] = struct{}{}
	}

	for _, i := range new {
		if _, ok := oldList[i]; !ok {
			added = append(added, i)
		}
	}

	return
}
