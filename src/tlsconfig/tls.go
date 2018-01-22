package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
)

// TLSCurveLookup is a lookup table for TLS Curve ID
var TLSCurveLookup = map[string]tls.CurveID{
	`curvep256`: tls.CurveP256,
	`curvep384`: tls.CurveP384,
	`curvep521`: tls.CurveP521,
	`x25519`:    tls.X25519,
}

// TLSVersionLookup is a lookup table for TLS Version ID
var TLSVersionLookup = map[string]uint16{
	`versionssl30`: tls.VersionSSL30,
	`versiontls10`: tls.VersionTLS10,
	`versiontls11`: tls.VersionTLS11,
	`versiontls12`: tls.VersionTLS12,
}

// TLSRenegotiateLookup is a lookup table for TLS renegotiate ID
var TLSRenegotiateLookup = map[string]tls.RenegotiationSupport{
	`renegotiatenever`:          tls.RenegotiateNever,
	`renegotiateonceasclient`:   tls.RenegotiateOnceAsClient,
	`renegotiatefreelyasclient`: tls.RenegotiateFreelyAsClient,
}

// TLSCipherLookup is a lookup table for TLS Cipher ID
var TLSCipherLookup = map[string]uint16{
	`tls_rsa_with_rc4_128_sha`:                tls.TLS_RSA_WITH_RC4_128_SHA,
	`tls_rsa_with_3des_ede_cbc_sha`:           tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	`tls_rsa_with_aes_128_cbc_sha`:            tls.TLS_RSA_WITH_AES_128_CBC_SHA,
	`tls_rsa_with_aes_256_cbc_sha`:            tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	`tls_rsa_with_aes_128_cbc_sha256`:         tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
	`tls_rsa_with_aes_128_gcm_sha256`:         tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	`tls_rsa_with_aes_256_gcm_sha384`:         tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	`tls_ecdhe_ecdsa_with_rc4_128_sha`:        tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
	`tls_ecdhe_ecdsa_with_aes_128_cbc_sha`:    tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	`tls_ecdhe_ecdsa_with_aes_256_cbc_sha`:    tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	`tls_ecdhe_rsa_with_rc4_128_sha`:          tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
	`tls_ecdhe_rsa_with_3des_ede_cbc_sha`:     tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
	`tls_ecdhe_rsa_with_aes_128_cbc_sha`:      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	`tls_ecdhe_rsa_with_aes_256_cbc_sha`:      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	`tls_ecdhe_ecdsa_with_aes_128_cbc_sha256`: tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
	`tls_ecdhe_rsa_with_aes_128_cbc_sha256`:   tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
	`tls_ecdhe_rsa_with_aes_128_gcm_sha256`:   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	`tls_ecdhe_ecdsa_with_aes_128_gcm_sha256`: tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	`tls_ecdhe_rsa_with_aes_256_gcm_sha384`:   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	`tls_ecdhe_ecdsa_with_aes_256_gcm_sha384`: tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	`tls_ecdhe_rsa_with_chacha20_poly1305`:    tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	`tls_ecdhe_ecdsa_with_chacha20_poly1305`:  tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	`tls_fallback_scsv`:                       tls.TLS_FALLBACK_SCSV,
}

// TLSConfig is user definable config for TLS
type TLSConfig struct {
	CertificateKey     string   `json:"certificatekey" toml:"certificatekey"`
	CertificateFile    string   `json:"certificatefile" toml:"certificatefile"`
	MinVersion         string   `json:"minversion" toml:"minversion"`
	MaxVersion         string   `json:"maxversion" toml:"maxversion"`
	Renegotiation      string   `json:"renegotiation" toml:"renegotiation"`
	CipherSuites       []string `json:"ciphersuites" toml:"ciphersuites"`
	CurvePreferences   []string `json:"curvepreferences" toml:"curvepreferences"`
	InsecureSkipVerify bool     `json:"insecureskipverify" toml:"insecureskipverify"`
}

// LoadCertificate loads the user definable config and returns the tls.Config
func LoadCertificate(t TLSConfig) (c *tls.Config, err error) {
	c = &tls.Config{}
	c.InsecureSkipVerify = t.InsecureSkipVerify
	c.PreferServerCipherSuites = true
	if t.MinVersion != "" {
		c.MinVersion = TLSVersionLookup[strings.ToLower(t.MinVersion)]
		if c.MinVersion == 0 {
			return c, fmt.Errorf("Unknown TLSMinVersion: %s", t.MinVersion)
		}
	}
	if t.MaxVersion != "" {
		c.MaxVersion = TLSVersionLookup[strings.ToLower(t.MaxVersion)]
		if c.MaxVersion == 0 {
			return c, fmt.Errorf("Unknown TLSMaxVersion: %s", t.MaxVersion)
		}
	}
	if t.Renegotiation != "" {
		c.Renegotiation = TLSRenegotiateLookup[strings.ToLower(t.Renegotiation)]
	}
	if len(t.CipherSuites) > 0 {
		for _, cipher := range t.CipherSuites {
			cn := TLSCipherLookup[strings.ToLower(cipher)]
			if cn == 0 {
				return c, fmt.Errorf("Unknown TLSCipher: %s", cipher)
			}
			c.CipherSuites = append(c.CipherSuites, cn)
		}
	}
	if len(t.CurvePreferences) > 0 {
		for _, curve := range t.CurvePreferences {
			cn := TLSCurveLookup[strings.ToLower(curve)]
			if cn == 0 {
				return c, fmt.Errorf("Unknown TLSCurve: %s", curve)
			}
			c.CurvePreferences = append(c.CurvePreferences, cn)
		}
	}

	if t.CertificateFile != "" && t.CertificateKey != "" {
		cert, err := tls.LoadX509KeyPair(t.CertificateFile, t.CertificateKey)
		if err != nil {
			return c, err
		}
		c.Certificates = []tls.Certificate{cert}
	}

	return c, nil
}

// AddCertificate adds a certificate to an existing config, based on TLSConfig
func AddCertificate(t TLSConfig, c *tls.Config) error {
	if t.CertificateFile != "" && t.CertificateKey != "" {
		cert, err := tls.LoadX509KeyPair(t.CertificateFile, t.CertificateKey)
		if err != nil {
			return err
		}

		var names []string

		details, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return fmt.Errorf("Failed to parse certificate for details")
		}
		if len(details.Subject.CommonName) > 0 {
			names = append(names, details.Subject.CommonName)
		}
		names = append(names, details.DNSNames...)

		// Check if we already loaded all dns names mentioned in this certificates
		existing := []string{}
		if c.NameToCertificate != nil {
			for dnsname := range c.NameToCertificate {
				existing = append(existing, dnsname)
			}
		}
		//fmt.Printf("Cert Add of names: %v\n", names)
		//fmt.Printf("Cert Existing names: %v\n", existing)
		//fmt.Printf("Cert Difference: %v\n", differenceArr(names, existing))
		if len(differenceArr(names, existing)) == 0 {
			fmt.Printf("Already loaded all the certificates in the new one, skipping load")
			//log.Fatal("bye\n")
		}
		//for _, knownCert := range c.Certificates {
		//fmt.Printf("Cert Add: %v\n", cert.Leaf.DNSNames)
		/*
			if reflect.DeepEqual(knownCert.Leaf.DNSNames, cert.Leaf.DNSNames) {
				return fmt.Errorf("Duplicate certificate defined in config: %v\n", cert.Leaf.DNSNames)
			}
		*/
		//}
		c.Certificates = append(c.Certificates, cert)
		c.BuildNameToCertificate()
	}

	return nil
}

// difference returns the elements in a that aren't in b
func differenceArr(a, b []string) []string {
	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}
	ab := []string{}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			ab = append(ab, x)
		}
	}
	return ab
}
