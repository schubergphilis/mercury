package dns

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	dnssrv "github.com/miekg/dns"
	"golang.org/x/crypto/ed25519"
	"gopkg.in/yaml.v2"
)

const (
	// ZoneSigningKey is the Zone Signing Key Type ID
	ZoneSigningKey uint16 = 256
	// KeySigningKey is the Key Signing Key Type ID
	KeySigningKey uint16 = 257
)

// Key is the type of provate key, which we read/store to disk
type Key struct {
	PrivateKey string `yaml:"privatekey"`
	privateKey crypto.PrivateKey
	Publish    time.Time     `yaml:"publish"`
	Activate   time.Time     `yaml:"activate"`
	Deactivate time.Time     `yaml:"deactivate"`
	Remove     time.Time     `yaml:"remove"`
	TTL        time.Duration `yaml:"ttl"`
	Algorithm  uint8         `yaml:"algorithm"`
	KeyTag     uint16        `yaml:"tag"`
}

// Keys is a collection of key for a domain
type Keys struct {
	Keys    map[string][]*Key // map[zone]Key
	KeyType uint16
}

// KeyStore is a collection of Key- and Zone signing Keys
type KeyStore struct {
	KeySigningKeys  Keys
	ZoneSigningKeys Keys
}

// NewKeyStore creates a new key store, that contains both KSK and ZSK
func NewKeyStore() *KeyStore {
	return &KeyStore{
		KeySigningKeys: Keys{
			KeyType: KeySigningKey,
			Keys:    make(map[string][]*Key),
		},
		ZoneSigningKeys: Keys{
			KeyType: ZoneSigningKey,
			Keys:    make(map[string][]*Key),
		},
	}
}

// Load all private keys that are not yet expired
func (k *KeyStore) Load(dir string) error {
	fmt.Printf("Loading keys from: %s\n", dir)
	if err := k.KeySigningKeys.load(dir + "/KSK"); err != nil {
		return err
	}
	if err := k.ZoneSigningKeys.load(dir + "/ZSK"); err != nil {
		return err
	}
	return nil
}

// load loads the keys of a zone
func (k *Keys) load(dir string) error {
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, "private") {

				zone, key, err := loadKey(path)
				if err != nil {
					log.Printf("%s failed to load", path)
				}
				if key != nil {

					k.Keys[zone] = append(k.Keys[zone], key)
				}
			}

			return nil
		})
	if err != nil {
		return err
	}
	return nil
}

// loadKey loads a key from disk
func loadKey(filepath string) (string, *Key, error) {
	base := path.Base(filepath)

	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", nil, err
	}

	key := &Key{}
	err = yaml.Unmarshal(data, &key)
	if err != nil {
		return "", nil, err
	}

	if key.Deactivate.Before(time.Now()) {
		log.Printf("%s expired", base)
		return "", nil, nil
	}

	switch key.Algorithm {
	//case dnssrv.RSASHA256:
	// key.privateKey =
	//case dnssrv.RSASHA512:
	case dnssrv.ECDSAP256SHA256:
		key.privateKey, err = ecdsaFromString(elliptic.P256(), key.PrivateKey)
		if err != nil {
			return "", nil, err
		}
		fmt.Printf("KEY: %+v", key.privateKey.(*ecdsa.PrivateKey).D)
	case dnssrv.ECDSAP384SHA384:
		key.privateKey, err = ecdsaFromString(elliptic.P384(), key.PrivateKey)
		if err != nil {
			return "", nil, err
		}
	default:
		return "", nil, fmt.Errorf("not a private key file")
	}

	zone := strings.Split(base, "+")[0]

	log.Printf("%s loaded", base)

	return zone, key, nil
}

// Save saves all keys to disk
func (k *KeyStore) Save(dir string) error {
	if err := k.KeySigningKeys.save(dir + "/KSK"); err != nil {
		return err
	}
	if err := k.ZoneSigningKeys.save(dir + "/ZSK"); err != nil {
		return err
	}
	return nil
}

// save saves all keys of a zone to disk
func (k *Keys) save(dir string) error {
	for zone, keys := range k.Keys {
		for _, key := range keys {
			err := key.save(k.KeyType, zone, dir)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// save saves a private key to disk
func (k *Key) save(keyType uint16, zone, dir string) error {
	dnskey := k.DNSKEY(keyType, zone)
	k.KeyTag = dnskey.KeyTag()
	fileName := fmt.Sprintf("%s/%s+%d+%d.private", dir, zone, k.Algorithm, dnskey.KeyTag())

	switch k.Algorithm {
	case dnssrv.ECDSAP256SHA256:
		k.PrivateKey = ecdsaToString(elliptic.P256(), k.privateKey.(*ecdsa.PrivateKey))
	case dnssrv.ECDSAP384SHA384:
		k.PrivateKey = ecdsaToString(elliptic.P384(), k.privateKey.(*ecdsa.PrivateKey))
	default:
		return fmt.Errorf("Unkown algorithm")
	}
	log.Printf("Saving file: %s", fileName)
	data, err := yaml.Marshal(k)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fileName, []byte(data), 0600)
}

// ecdsaToString converts a ECDSA key to string
func ecdsaToString(c elliptic.Curve, key *ecdsa.PrivateKey) string {
	return toBase64(intToBytes(key.D, c.Params().BitSize/8))
}

// ecdsaFromString converts a string to a ECDSA key
func ecdsaFromString(c elliptic.Curve, der string) (*ecdsa.PrivateKey, error) {
	p := new(ecdsa.PrivateKey)
	p.D = new(big.Int)

	b, err := fromBase64([]byte(der))
	if err != nil {
		return nil, err
	}
	p.D.SetBytes(b)
	p.Curve = c
	p.PublicKey.X, p.PublicKey.Y = c.ScalarBaseMult(p.D.Bytes())
	return p, nil
}

/*
// ecdsaToString converts a ECDSA key to string
func ecdsaToStrings(privateKey *ecdsa.PrivateKey) (string, string) {
	x509Encoded, _ := x509.MarshalECPrivateKey(privateKey)
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})

	x509EncodedPub, _ := x509.MarshalPKIXPublicKey(privateKey.PublicKey)
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	return string(pemEncoded), string(pemEncodedPub)

	//e := eccp.Marshal(c, key.X, key.Y)
	//return toBase64(e)
}

func ecdsaFromString(pemEncoded, pemEncodedPub string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemEncoded))
	x509Encoded := block.Bytes
	privateKey, _ := x509.ParseECPrivateKey(x509Encoded)

	blockPub, _ := pem.Decode([]byte(pemEncodedPub))
	x509EncodedPub := blockPub.Bytes
	genericPublicKey, _ := x509.ParsePKIXPublicKey(x509EncodedPub)
	publicKey := genericPublicKey.(*ecdsa.PublicKey)

	privateKey.PublicKey = *publicKey
	return privateKey, nil

}
*/

/*
// ecdsaFromString converts a string to a ECDSA key
func ecdsaFromString(c elliptic.Curve, der string) (*ecdsa.PrivateKey, error) {
	b, err := fromBase64([]byte(der))
	if err != nil {
		return nil, err
	}
	x, y := eccp.Unmarshal(c, b)
	k := new(ecdsa.PrivateKey)
	k.X = x
	k.Y = y
	k.Curve = c
	return k, nil
}

*/

/*
// ecdsaToString converts a ECDSA key to string
func ecdsaToString(c elliptic.Curve, key *ecdsa.PrivateKey) string {
	e := eccp.Marshal(c, key.X, key.Y)
	return toBase64(e)
}

// ecdsaFromString converts a string to a ECDSA key
func ecdsaFromString(c elliptic.Curve, der string) (*ecdsa.PrivateKey, error) {
	b, err := fromBase64([]byte(der))
	if err != nil {
		return nil, err
	}
	x, y := eccp.Unmarshal(c, b)
	k := new(ecdsa.PrivateKey)
	k.X = x
	k.Y = y
	k.Curve = c
	return k, nil
}
*/

/*
Private-key-format: v1.3
Algorithm: 13 (ECDSAP256SHA256)
PrivateKey: fGMaaDAYWA7BMovxIUJ2q8MpJSUjaUKzccumxJ+BX/E=
*/

/*
type PrivateKeyFile struct {
	PrivateKeyFormat string `yaml:"Private-key-format"`
	Algorithm        string `yaml:"Algorithm"`
	PrivateKey       string `yaml:"PrivateKey"`
}
*/

// NewPrivateKey creates a new private key
func NewPrivateKey(keyType uint16, algorithm uint8) (*Key, error) {
	k := &Key{}

	k.Algorithm = algorithm
	switch k.Algorithm {
	case dnssrv.ECDSAP256SHA256:
		c := elliptic.P256()
		priv, err := ecdsa.GenerateKey(c, rand.Reader)
		if err != nil {
			return nil, err
		}
		k.privateKey = priv
	case dnssrv.ECDSAP384SHA384:
		c := elliptic.P384()
		priv, err := ecdsa.GenerateKey(c, rand.Reader)
		if err != nil {
			return nil, err
		}
		k.privateKey = priv
	case dnssrv.RSASHA512, dnssrv.RSASHA256:
		var keySize int
		if keyType == KeySigningKey {
			keySize = 4096
		}
		if keyType == ZoneSigningKey {
			keySize = 1024
		}
		priv, err := rsa.GenerateKey(rand.Reader, keySize)
		if err != nil {
			return nil, err
		}
		k.privateKey = priv
	default:
		return nil, fmt.Errorf("invalid algorithm: %s", dnssrv.AlgorithmToString[algorithm])
	}

	return k, nil
}

// SetRollover adds a new key to the rollover mechanism
func (k *KeyStore) SetRollover(keyType uint16, zone string, ttl time.Duration, key *Key) error {
	if !strings.HasSuffix(zone, ".") {
		zone += "."
	}
	if keyType == KeySigningKey {
		return k.KeySigningKeys.setRollover(zone, ttl, key)
	}
	if keyType == ZoneSigningKey {
		return k.ZoneSigningKeys.setRollover(zone, ttl, key)
	}
	return nil
}

// setRollover adds a key to the rollover and sets the rollover values
func (k *Keys) setRollover(zone string, ttl time.Duration, key *Key) error {
	key.TTL = ttl
	now := time.Now()
	if _, ok := k.Keys[zone]; !ok {
		// new zone
		key.Publish = now
		key.Activate = now
		key.Deactivate = now.Add(ttl)
		// key.Remove
		k.Keys[zone] = append(k.Keys[zone], key)
		return nil
	}

	// existing zone
	var lastActive time.Time
	var lastActiveKey *Key
	for _, k := range k.Keys[zone] {
		if k.Deactivate.After(lastActive) {
			lastActive = k.Deactivate
			lastActiveKey = k
		}
	}
	key.Publish = lastActiveKey.Activate
	key.Activate = lastActiveKey.Deactivate
	key.Deactivate = key.Activate.Add(ttl)

	lastActiveKey.Remove = key.Deactivate

	k.Keys[zone] = append(k.Keys[zone], key)
	return nil
}

// DNSKEYS returns the DNSKEY responses for a zone of a specific type
func (k *Keys) DNSKEYS(keyType uint16, zone string, tm time.Time) []*dnssrv.DNSKEY {
	var dnskeys []*dnssrv.DNSKEY
	if _, ok := k.Keys[zone]; !ok {
		return dnskeys
	}

	for _, k := range k.Keys[zone] {
		if k.Publish.Before(tm) && k.Deactivate.After(tm) {
			dnskey := k.DNSKEY(keyType, zone)
			dnskeys = append(dnskeys, dnskey)
		}

		//log.Printf("DS: %s", dnskey.ToDS(dnssrv.SHA256))

	}
	return dnskeys
}

// DNSKEY returns a DNSKEY record for a specific key
func (k *Key) DNSKEY(keyType uint16, zone string) *dnssrv.DNSKEY {
	dnskey := new(dnssrv.DNSKEY)
	dnskey.Hdr.Rrtype = dnssrv.TypeDNSKEY
	dnskey.Hdr.Name = zone
	dnskey.Hdr.Class = dnssrv.ClassINET
	dnskey.Hdr.Ttl = uint32(k.TTL / time.Second)
	dnskey.Flags = keyType
	dnskey.Protocol = 3 // always 3

	switch k.Algorithm {
	case dnssrv.ECDSAP256SHA256, dnssrv.ECDSAP384SHA384:
		priv := k.privateKey.(*ecdsa.PrivateKey)
		dnskey.PublicKey = publicKeyECDSA(k.Algorithm, priv.PublicKey.X, priv.PublicKey.Y)
	case dnssrv.RSASHA512, dnssrv.RSASHA256:
		priv := k.privateKey.(*rsa.PrivateKey)
		dnskey.PublicKey = publicKeyRSA(priv.PublicKey.E, priv.PublicKey.N)
	}
	return dnskey
}

// DNSKEYS returns all DNSKEY for a zone
func (k *KeyStore) DNSKEYS(zone string, tm time.Time) []*dnssrv.DNSKEY {
	if !strings.HasSuffix(zone, ".") {
		zone += "."
	}
	var dnskeys []*dnssrv.DNSKEY
	dnskeys = append(dnskeys, k.KeySigningKeys.DNSKEYS(KeySigningKey, zone, tm)...)
	dnskeys = append(dnskeys, k.ZoneSigningKeys.DNSKEYS(ZoneSigningKey, zone, tm)...)
	return dnskeys
}

// algorithm numbers
// https://www.iana.org/assignments/dns-sec-alg-numbers/dns-sec-alg-numbers.xhtml
// digent type numbers
// 1  SHA-1
// 2  SHA-256
// 4  SHA-384

// Set the public key (the value E and N)
func publicKeyRSA(_E int, _N *big.Int) string {
	if _E == 0 || _N == nil {
		return ""
	}
	buf := exponentToBuf(_E)
	buf = append(buf, _N.Bytes()...)
	return toBase64(buf)
}

// Set the public key for Elliptic Curves
func publicKeyECDSA(a uint8, _X, _Y *big.Int) string {
	if _X == nil || _Y == nil {
		return ""
	}
	var intlen int
	switch a {
	case dnssrv.ECDSAP256SHA256:
		intlen = 32
	case dnssrv.ECDSAP384SHA384:
		intlen = 48
	}
	return toBase64(curveToBuf(_X, _Y, intlen))
}

// Set the public key for Ed25519
func publicKeyED25519(_K ed25519.PublicKey) string {
	if _K == nil {
		return ""
	}
	return toBase64(_K)
}

// Set the public key (the values E and N) for RSA
// RFC 3110: Section 2. RSA Public KEY Resource Records
func exponentToBuf(_E int) []byte {
	var buf []byte
	i := big.NewInt(int64(_E)).Bytes()
	if len(i) < 256 {
		buf = make([]byte, 1, 1+len(i))
		buf[0] = uint8(len(i))
	} else {
		buf = make([]byte, 3, 3+len(i))
		buf[0] = 0
		buf[1] = uint8(len(i) >> 8)
		buf[2] = uint8(len(i))
	}
	buf = append(buf, i...)
	return buf
}

// Set the public key for X and Y for Curve. The two
// values are just concatenated.
func curveToBuf(_X, _Y *big.Int, intlen int) []byte {
	buf := intToBytes(_X, intlen)
	buf = append(buf, intToBytes(_Y, intlen)...)
	return buf
}

var base32HexNoPadEncoding = base32.HexEncoding.WithPadding(base32.NoPadding)

func fromBase32(s []byte) (buf []byte, err error) {
	for i, b := range s {
		if b >= 'a' && b <= 'z' {
			s[i] = b - 32
		}
	}
	buflen := base32HexNoPadEncoding.DecodedLen(len(s))
	buf = make([]byte, buflen)
	n, err := base32HexNoPadEncoding.Decode(buf, s)
	buf = buf[:n]
	return
}

func toBase32(b []byte) string {
	return base32HexNoPadEncoding.EncodeToString(b)
}

func fromBase64(s []byte) (buf []byte, err error) {
	buflen := base64.StdEncoding.DecodedLen(len(s))
	buf = make([]byte, buflen)
	n, err := base64.StdEncoding.Decode(buf, s)
	buf = buf[:n]
	return
}

func toBase64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

// Helper function for packing and unpacking
func intToBytes(i *big.Int, length int) []byte {
	buf := i.Bytes()
	if len(buf) < length {
		b := make([]byte, length)
		copy(b[length-len(buf):], buf)
		return b
	}
	return buf
}
