package tlsconfig

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/schubergphilis/mercury/pkg/logging"

	"golang.org/x/crypto/ocsp"
)

// OCSPHandler refreshes OCSP staple if expired or not present
func OCSPHandler(c *tls.Config, quit chan bool) {
	log := logging.For("tlsconfig/ocsp/handler").WithField("server", c.ServerName)
	expiry, err := RenewOCSP(c)
	if err != nil {
		log.WithField("renew", fmt.Sprintf("%s", expiry)).WithError(err).Warn("Initial OCSP get failed")
	} else {
		log.WithField("renew", fmt.Sprintf("%s", expiry)).Info("Initial OCSP get succesfull")
	}

	ticker := time.NewTicker(expiry.Sub(time.Now()))
	for {
		select {
		case <-ticker.C:
			expiry, err := RenewOCSP(c)
			if err != nil {
				log.WithField("renew", fmt.Sprintf("%s", expiry)).WithError(err).Warn("OCSP renewal failed")
			} else {
				log.WithField("renew", fmt.Sprintf("%s", expiry)).Info("OCSP renewal succesfull")
			}
			ticker = time.NewTicker(expiry.Sub(time.Now()))
		case <-quit:
			return
		}
	}
}

// RenewOCSP renews the OCSP reply
// Caveat - the expiry time is that of the shortest certificate
func RenewOCSP(c *tls.Config) (time.Time, error) {
	expire := time.Now().Add(24 * 7 * time.Hour) // refresh every week
	for cid, certs := range c.Certificates {
		var certificates []*x509.Certificate
		for _, subcert := range certs.Certificate {
			cert, err := x509.ParseCertificate(subcert)
			if err != nil {
				continue // skip parsed unparsable certificates, see if we can find more
			}

			certificates = append(certificates, cert)
		}

		OCSPStaple, OCSPResponse, err := GetOCSPResult(certificates)
		if err != nil {
			expire = time.Now().Add(30 * time.Minute) // temporary failure?, try again in an hour
			return expire, fmt.Errorf("OCSP Result failed:%s", err)
			// fail here, so we don't load half working ocsp staples
		}

		if OCSPResponse.NextUpdate.Before(expire) { // if expiry is shorter, update before this
			expire = OCSPResponse.NextUpdate.Add(-6 * time.Hour) // take 6 hours off expirey to allow time for renew
			if expire.Before(time.Now()) {
				expire = time.Now().Add(1 * time.Hour)
			}
		}

		c.Certificates[cid].OCSPStaple = OCSPStaple
	}
	return expire, nil
}

// GetOCSPResult collects the required certificates and handles the OCSP call
func GetOCSPResult(certificates []*x509.Certificate) ([]byte, *ocsp.Response, error) {
	var HTTPClient = http.Client{Timeout: 10 * time.Second}

	issuedCert := certificates[0]
	if len(issuedCert.OCSPServer) == 0 {
		return nil, nil, fmt.Errorf("no OCSP server specified in cert")
	}
	//fmt.Printf("LEN:%d CA?:%t CommonName:%s DNS:%v\n", len(certificates), issuedCert.IsCA, issuedCert.Subject.CommonName, issuedCert.DNSNames)
	var issuerCert *x509.Certificate
	if len(certificates) == 1 { // self signed? lets see if we can find the cert
		if len(issuedCert.IssuingCertificateURL) == 0 {
			return nil, nil, fmt.Errorf("no issuing certificate URL")
		}

		req, err := http.NewRequest("GET", issuedCert.IssuingCertificateURL[0], nil)
		if err != nil {
			return nil, nil, err
		}

		res, err := HTTPClient.Do(req)
		if err != nil {
			return nil, nil, err
		}

		defer res.Body.Close()
		issuerBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, nil, err
		}

		issuerCert, err = x509.ParseCertificate(issuerBytes)
		if err != nil {
			return nil, nil, err
		}

	} else {
		for _, cert := range certificates {
			if cert.IsCA {
				issuerCert = cert
			}
		}
	}

	if issuedCert == nil {
		return nil, nil, fmt.Errorf("Could not locate CA certificate for OCSP checking")
	}

	if issuerCert == nil {
		return nil, nil, fmt.Errorf("Could not locate CA certificate issuer for OCSP checking, no CA cert included?")
	}

	// Create OCSP request
	ocspRequest, err := ocsp.CreateRequest(issuedCert, issuerCert, nil)
	if err != nil {
		return nil, nil, err
	}

	// Post request
	reader := bytes.NewReader(ocspRequest)
	req, err := http.NewRequest("POST", issuedCert.OCSPServer[0], reader)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/ocsp-request")
	res, err := HTTPClient.Do(req)

	if err != nil {
		return nil, nil, err
	}

	defer req.Body.Close()

	// Read Request
	ocspResult, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	// Check response
	ocspResultStatus, err := ocsp.ParseResponse(ocspResult, issuerCert)
	if err != nil {
		return nil, nil, err
	}

	return ocspResult, ocspResultStatus, nil
}
