package tlsconfig

import (
	"testing"
)

func TestTLSConfig(t *testing.T) {
	config := TLSConfig{
		MinVersion:       "VersionTLS10",
		MaxVersion:       "VersionTLS12",
		Renegotiation:    "RenegotiateNever", // doesn't fail if incorrect, defaults back to never renegotiatie
		CipherSuites:     []string{"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384", "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"},
		CurvePreferences: []string{"CurveP256"},
		CertificateKey:   "../../build/test/ssl/self_signed_certificate.key",
		CertificateFile:  "../../build/test/ssl/self_signed_certificate.crt",
	}
	c, err := LoadCertificate(config)
	if err != nil {
		t.Errorf("Error parsing TLS Config: %s", err)
	}

	fail := config
	fail.MinVersion = "NoMin"
	_, err = LoadCertificate(fail)
	if err == nil {
		t.Errorf("Expected TLS Parsing error for MinVersion")
	}

	fail = config
	fail.MaxVersion = "NoMax"
	_, err = LoadCertificate(fail)
	if err == nil {
		t.Errorf("Expected TLS Parsing error for MaxVersion: %+v", c)
	}

	fail = config
	fail.CipherSuites = append(fail.CipherSuites, "NoCipher")
	_, err = LoadCertificate(fail)
	if err == nil {
		t.Errorf("Expected TLS Parsing error for CipherSuites")
	}

	fail = config
	fail.CurvePreferences = append(fail.CurvePreferences, "NoCurve")
	_, err = LoadCertificate(fail)
	if err == nil {
		t.Errorf("Expected TLS Parsing error for CurvePreferences")
	}

	fail = config
	fail.CertificateFile = "nonexisting"
	_, err = LoadCertificate(fail)
	if err == nil {
		t.Errorf("Expected File load error")
	}

	err = AddCertificate(config, c)
	if err != nil {
		t.Errorf("Expected Add certificate to work")
	}

	fail = config
	fail.CertificateFile = "nonexisting"
	err = AddCertificate(fail, c)
	if err == nil {
		t.Errorf("Expected Add certificate fail loading this certificate")
	}

	// TODO: write a test
	RenewOCSP(c)
	//_, err = RenewOCSP(c)
	/*
		if err == nil {
			t.Errorf("Unless your loading a valid certificate, this should have failed. err=%s", err)
		} else if err.Error() != "no OCSP server specified in cert" {
			t.Errorf("OCSP expected no server specified in cert. got:%s", err)
		}
	*/

}
