package healthcheck

import (
	"encoding/json"
	"testing"
)

type TestStatus struct {
	Status Status
}

func TestStatusJson(t *testing.T) {
	s := TestStatus{
		Status: Online,
	}

	j, err := json.Marshal(s)
	if err != nil {
		t.Errorf("Marshal failed: %s", err)
	}

	tmp := &TestStatus{}

	err = json.Unmarshal(j, tmp)
	if err != nil {
		t.Errorf("UnMarshal failed: %s", err)
	}

}
