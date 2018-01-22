package check

import (
	"encoding/json"
	"fmt"

	"github.com/schubergphilis/mercury/internal/config"
	"github.com/schubergphilis/mercury/pkg/logging"
)

// checkBackendsOnline checks if all backends are online
func checkBackendsOnline(pools map[string]config.LoadbalancePool) (int, error) {
	var faultyTargets []string
	for poolname, pool := range pools {
		for backendname, backend := range pool.Backends {
			offline := 0
			online := 0
			for _, node := range backend.Nodes {
				if node.Online == false {
					offline++
				} else {
					online++
				}
			}
			// Report error nodes if any node is offline, or if active/passive has none online - with the exception of active/passive with only 1 node - we ignore this
			if (offline > 0 && backend.BalanceMode.ActivePassive != YES) || (offline > 1 && online == 0 && backend.BalanceMode.ActivePassive == YES) {
				for _, node := range backend.Nodes {
					if node.Online == false {
						faultyTargets = append(faultyTargets, fmt.Sprintf("Node:%s:%d (Backend:%s Pool:%s)", node.IP, node.Port, backendname, poolname))
					}
				}
			}
		}
	}
	if faultyTargets != nil {
		return CRITICAL, fmt.Errorf("The following node(s) failed their healthcheck(s) and are Offline: %v", faultyTargets)
	}
	return OK, nil
}

// checkBackendsOnline checks if all backends are online
func checkBackendsHasNodes(pools map[string]config.LoadbalancePool) (int, error) {
	var faultyTargets []string
	for poolname, pool := range pools {
		for backendname, backend := range pool.Backends {
			nodes := 0
			for _, node := range backend.Nodes {
				if node.Online == true {
					nodes++
				}
			}
			/*
				BUG: for some reason 1 node offline, doesn't hit the backend down message it seems?
			*/
			if backend.BalanceMode.ActivePassive == YES {
				if nodes == 0 && len(backend.Nodes) > 1 {
					faultyTargets = append(faultyTargets, fmt.Sprintf("(Backend:%s (Pool:%s)", backendname, poolname))
				}
			} else if backend.ConnectMode != "internal" && nodes == 0 {
				faultyTargets = append(faultyTargets, fmt.Sprintf("(Backend:%s (Pool:%s)", backendname, poolname))
			}
		}
	}
	if faultyTargets != nil {
		return CRITICAL, fmt.Errorf("The following backend(s) have NO nodes available and are Offline: %v", faultyTargets)
	}
	return OK, nil
}

// Backend checks backend status
func Backend() int {
	log := logging.For("check/glb")
	body, err := GetBody(fmt.Sprintf("https://%s:%d/backend", config.Get().Web.Binding, config.Get().Web.Port))
	if err != nil {
		fmt.Printf("Error connecting to Mercury at %s:%d. Is the service running? (error:%s)\n", config.Get().Web.Binding, config.Get().Web.Port, err)
		return CRITICAL
	}
	var loadbalancer config.Loadbalancer
	err = json.Unmarshal(body, &loadbalancer)
	if err != nil {
		fmt.Printf("Error parsing json given by the Mercury service: %s\n", err)
		return CRITICAL
	}
	// Prepare data
	var criticals []string
	var warnings []string
	// Execute Checks
	log.Debug("Checking if backend has atleast 1 node online")
	if exitcode, err := checkBackendsHasNodes(loadbalancer.Pools); err != nil {
		switch exitcode {
		case CRITICAL:
			criticals = append(criticals, err.Error())
		case WARNING:
			warnings = append(warnings, err.Error())
		}
	}
	log.Debug("Checking if all backend nodes are online")
	if exitcode, err := checkBackendsOnline(loadbalancer.Pools); err != nil {
		switch exitcode {
		case CRITICAL:
			criticals = append(criticals, err.Error())
		case WARNING:
			warnings = append(warnings, err.Error())
		}
	}
	if len(criticals) > 0 {
		fmt.Printf("CRITICAL: %+v\n", criticals)
		return CRITICAL
	}
	if len(warnings) > 0 {
		fmt.Printf("WARNING: %v\n", warnings)
		return WARNING
	}
	fmt.Println("OK: All checks are fine!")
	return OK
}
