package proxy

import "testing"
import "github.com/stretchr/testify/assert"

func TestIPConversion(t *testing.T) {
	client := stringToClientIP("10.11.12.13")
	assert.Equal(t, "10.11.12.13", client.IP)
	assert.Equal(t, 0, client.Port)

	client = stringToClientIP("10.11.12.13:23")
	assert.Equal(t, "10.11.12.13", client.IP)
	assert.Equal(t, 23, client.Port)

	client = stringToClientIP("fe80::493d:d242:9475:a510:23")
	assert.Equal(t, "fe80::493d:d242:9475:a510", client.IP)
	assert.Equal(t, 23, client.Port)
}
