package param

/*
 * param package handles the cli parameters
 */

import (
	"flag"
	"sync"
)

// Config is the cmd parameter output
type Config struct {
	ConfigFile          *string
	PidFile             *string
	CheckGLB            *bool
	CheckConfig         *bool
	CheckBackend        *bool
	Debug               *bool
	Version             *bool
	BackendName         *string
	PoolName            *string
	DNSName             *string
	ClusterOnly         *bool
	SigningKeyTTL       *int
	CreateKeySigningKey *bool
	ReadSigningKey      *bool
	SigningZone         *string
	SigningAlgorithm    *string
	KeyDir              *string
}

var (
	config     *Config
	configLock sync.RWMutex
)

func init() {
	c := Config{
		ConfigFile:          flag.String("config-file", "../../test/mercury.toml", "path to your mercury toml confg file"),
		PidFile:             flag.String("pid-file", "/run/mercury.pid", "path to your pid file"),
		CheckGLB:            flag.Bool("check-glb", false, "gives you a GLB report"),
		CheckConfig:         flag.Bool("check-config", false, "does a config check"),
		CheckBackend:        flag.Bool("check-backend", false, "gives you a Backend report"),
		Debug:               flag.Bool("debug", false, "force logging to debug mode"),
		Version:             flag.Bool("version", false, "display version"),
		BackendName:         flag.String("backend-name", "", "only check selected backend name"),
		PoolName:            flag.String("pool-name", "", "only check selected pool name"),
		DNSName:             flag.String("dns-name", "", "only check selected dns name"),
		ClusterOnly:         flag.Bool("cluster-only", false, "only check cluster"),
		SigningKeyTTL:       flag.Int("signing-key-ttl", 86400*365, "ttl to keep the Key Signing Key, not that you must manually create a new one after this ttl (time in seconds)"),
		CreateKeySigningKey: flag.Bool("create-key-signing-key", false, "creates a key to sign a zone keys (you need at least 1 of these for many zones)"),
		ReadSigningKey:      flag.Bool("read-signing-key", false, "reads the signing key and provides the matching DS and DNSKEY records"),
		SigningZone:         flag.String("signing-zone", "", "zone to sign with the key"),
		SigningAlgorithm:    flag.String("signing-algorithm", "ECDSAP256SHA256", "algorithm to use when signing/creating keys (RSASHA256, RSASHA512, ECDSAP256SHA256, ECDSAP348SHA348)"),
		KeyDir:              flag.String("key-dir", "", "dir to find the signatures"),
	}
	flag.Parse()
	config = &c
}

// Get Allows you to get a parameter
func Get() *Config {
	configLock.RLock()
	defer configLock.RUnlock()
	return config
}

// SetConfig sets the config file
func SetConfig(file string) {
	configLock.Lock()
	defer configLock.Unlock()
	config.ConfigFile = &file
}
