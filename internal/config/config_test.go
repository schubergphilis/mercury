package config

import (
	"testing"

	"github.com/schubergphilis/mercury/pkg/logging"
	"github.com/schubergphilis/mercury/pkg/param"
)

func TestConfig(t *testing.T) {
	t.Logf("TestConfig...")
	logging.Configure("stdout", "info")
	if err := LoadConfig("nonexistantfile"); err == nil {
		t.Errorf("Expected error on loading false config (got:%s)", err)
	}
	if err := LoadConfig("../../build/test/broken-config.toml"); err == nil {
		t.Errorf("Expected error on loading false config (got:%s)", err)
	}

	if err := LoadConfig("../../build/test/second-config.toml"); err != nil {
		t.Errorf("Expected error on loading false config (got:%s)", err)
	}

	if Get().DNS.Binding != "localhost" {
		t.Errorf("Expected DNS binding to be test-second-config (got:%s)", Get().DNS.Binding)
	}

	param.SetConfig("../../build/test/broken-config.toml")
	ReloadConfig()

	if Get().DNS.Binding != "localhost" {
		t.Errorf("Expected DNS binding to be test-second-config (got:%s)", Get().DNS.Binding)
	}

	param.SetConfig("../../build/test/second-config.toml")
	ReloadConfig()

	// Send this process a SIGHUP with new config
	/*
		param.SetConfig("../../test/second-config.toml")
		t.Logf("sighup...")
		syscall.Kill(syscall.Getpid(), syscall.SIGHUP)
		time.Sleep(1 * time.Second)

		if Get().DNS.Binding != "localhost" {
			t.Errorf("Expected DNS binding to be localhost (got:%s)", Get().DNS.Binding)
		}
	*/
	Lock()
	config.DNS.Binding = "test"
	Unlock()

	if Get().DNS.Binding != "test" {
		t.Errorf("Expected DNS binding to be test (got:%s)", Get().DNS.Binding)
	}

	// send SIGHUP with faulty config
	/*
		param.SetConfig("../../test/broken-config")
		t.Logf("sighup...")
		syscall.Kill(syscall.Getpid(), syscall.SIGHUP)
		time.Sleep(1 * time.Second)

		t.Errorf("BREAK1")
		return
	*/

}
