package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIpv4Subnet(t *testing.T) {
	ips := map[string]string{
		"127.0.0.1":                "127.0.0.1/32",
		"255.255.255.255":          "255.255.255.255/32",
		"::1":                      "::1/128",
		"fe80::2e0:81ff:fed4:36f5": "fe80::2e0:81ff:fed4:36f5/128",
	}

	for ip, result := range ips {
		r := addSubnet(ip)
		assert.Equal(t, result, r)
	}
}
