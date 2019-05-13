package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceAddedAndDeleted(t *testing.T) {
	old := []string{"one", "two", "three"}
	new := []string{"two", "three", "four"}

	added, deleted := sliceAddedAndDeleted(old, new)
	assert.Equal(t, added[0], "four")
	assert.Equal(t, deleted[0], "one")
}
