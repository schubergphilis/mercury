package check

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	// OK is ok value
	OK = 0
	// WARNING is warning value
	WARNING = 1
	// CRITICAL is critical value
	CRITICAL = 2
	// UNKNOWN is unknown value
	UNKNOWN = 3
	// YES when yes simply isn't enough
	YES = "yes"
)

// GetBody Returns the body of a request
func GetBody(url string) ([]byte, error) {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error reading status: %s", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading body: %s", err)
	}
	return body, nil
}
