package dns

import (
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/schubergphilis/mercury/pkg/balancer"
	"github.com/schubergphilis/mercury/pkg/param"
)

type NewKeyChannel struct {
}

// InitKeyStore initializes the keystore
func InitKeyStore(dir string, keyUpdates chan NewKeyChannel) {
	dnsmanager.Lock()
	defer dnsmanager.Unlock()
	dnsmanager.KeyStore.Load(dir)
	go dnssecAddRecords(keyUpdates)
}

func dnssecAddRecords(keyUpdates chan NewKeyChannel) {
	var zones []string
	// go through all key signing keys
	for zone, keys := range dnsmanager.KeyStore.KeySigningKeys.Keys {
		hasRecords := false
		for _, key := range keys {
			if key.Activate.After(time.Now()) {
				continue
			}
			DNSKEY := key.DNSKEY(KeySigningKey, zone)

			// DS
			hash := sha256.New()
			hash.Write([]byte(fmt.Sprintf("%s-%s-%x-%s", zone, "", "DS", DNSKEY.ToDS(2))))
			uuid := fmt.Sprintf("%x", hash.Sum(nil))
			ds := Record{
				Name:       "",
				Type:       "DS",
				Target:     trimleft(DNSKEY.ToDS(2).String(), "DS"),
				TTL:        int(key.TTL) / int(time.Second),
				Local:      true,
				UUID:       uuid,
				Statistics: balancer.NewStatistics(uuid, 0),
			}
			AddLocalRecord(nodot(zone), ds)

			// DNSKEY
			hash = sha256.New()
			hash.Write([]byte(fmt.Sprintf("%s-%s-%x-%s", zone, "", "DNSKEY", DNSKEY.ToDS(2))))
			uuid = fmt.Sprintf("%x", hash.Sum(nil))
			dnskey := Record{
				Name:       "",
				Type:       "DNSKEY",
				Target:     trimleft(DNSKEY.String(), "DNSKEY"),
				TTL:        int(key.TTL) / int(time.Second),
				Local:      true,
				UUID:       uuid,
				Statistics: balancer.NewStatistics(uuid, 0),
			}
			AddLocalRecord(nodot(zone), dnskey)

			hasRecords = true
		}
		if hasRecords == true {
			zones = append(zones, zone)
		}
	}

	// go through zone signing keys for each zone
	for _, zone := range zones {
		keys, ok := dnsmanager.KeyStore.ZoneSigningKeys.Keys[zone]
		if !ok || len(keys) == 0 {
			// we have no zone signing keys yet, we need to generate one
			log.Printf("Creating ZSK for %s", zone)
			key, err := NewPrivateKey(ZoneSigningKey, dnsmanager.KeyStore.KeySigningKeys.Keys[zone][0].Algorithm)
			if err != nil {
				log.Printf("Error creating ZSK for %s: %s", zone, err)
				continue
			}
			dnsmanager.KeyStore.SetRollover(ZoneSigningKey, zone, dnsmanager.KeyStore.KeySigningKeys.Keys[zone][0].TTL, key)
			dnsmanager.KeyStore.Save(*param.Get().KeyDir)
		}
		if keys, ok := dnsmanager.KeyStore.ZoneSigningKeys.Keys[zone]; ok {
			for _, key := range keys {

				DNSKEY := key.DNSKEY(ZoneSigningKey, zone)
				// DNSKEY
				hash := sha256.New()
				hash.Write([]byte(fmt.Sprintf("%s-%s-%x-%s", zone, "", "DNSKEY", DNSKEY.ToDS(2))))
				uuid := fmt.Sprintf("%x", hash.Sum(nil))
				dnskey := Record{
					Name:       "",
					Type:       "DNSKEY",
					Target:     trimleft(DNSKEY.String(), "DNSKEY"),
					TTL:        int(key.TTL) / int(time.Second),
					Local:      true,
					UUID:       uuid,
					Statistics: balancer.NewStatistics(uuid, 0),
				}
				AddLocalRecord(nodot(zone), dnskey)
			}
		}
	}

}

func nodot(s string) string {
	if strings.HasSuffix(s, ".") {
		return strings.TrimRight(s, ".")
	}
	return s
}

func trimleft(s, sub string) string {
	i := strings.Index(s, sub)
	if i > 0 {
		return s[i+len(sub):]
	}
	return s
}
