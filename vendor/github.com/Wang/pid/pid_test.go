package pid

import (
	"fmt"
	"os"
	"testing"
)

func TestPidGetValue(t *testing.T) {
	Debug = true
	if pid, err := GetValue(""); err == nil || pid != 0 {
		t.Error("get not exist pid file")
	}
	f, _ := os.Create("./test.pid")
	f.WriteString("12345s")
	f.Close()
	if pid, err := GetValue("test.pid"); err == nil || pid != 0 {
		t.Error("get not exist pid file")
	}
	os.RemoveAll("./test.pid")

	f, _ = os.Create("./test.pid")
	f.WriteString("12345")
	f.Close()
	if pid, err := GetValue("test.pid"); err != nil || pid == 0 {
		t.Error("get not exist pid file")
	}
	os.RemoveAll("./test.pid")
}

func TestCreate(t *testing.T) {
	Debug = true
	if pid, err := Create("my.pid"); err != nil {
		t.Error(err.Error())
	} else {
		fmt.Printf("pid: %d\n", pid)
	}
	if pid, err := Create("my.pid"); err != ErrPidExists {
		t.Error(err.Error())
	} else {
		fmt.Printf("pid: %d\n", pid)
	}
	if pid, err := GetValue("my.pid"); err != nil {
		t.Error(err.Error())
	} else {
		if ok, err := IsActive(pid); !ok {
			t.Error(err.Error())
		}
	}
}
