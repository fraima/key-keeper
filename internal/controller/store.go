package controller

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

func (s *controller) storeKey(path string, privare, public []byte) error {
	if err := os.WriteFile(path+".pem", privare, 0600); err != nil {
		return fmt.Errorf("failed to save privare key with path %s: %w", path, err)
	}

	if err := os.WriteFile(path+".pub", public, 0600); err != nil {
		return fmt.Errorf("failed to public key file: %w", err)
	}
	return nil
}

func (s *controller) storeCertificate(path string, crt, key []byte) error {
	if crt != nil {
		if err := os.WriteFile(path+".pem", crt, 0644); err != nil {
			return fmt.Errorf("failed to save certificate with path %s: %w", path, err)
		}
	}

	if key != nil {
		if err := os.WriteFile(path+"-key.pem", key, 0600); err != nil {
			return fmt.Errorf("failed to save key file: %w", err)
		}
	}
	return nil
}

func (s *controller) readCertificate(path string) (*tls.Certificate, error) {
	crt, err := os.ReadFile(path + ".pem")
	if err != nil {
		return nil, err
	}

	key, err := os.ReadFile(path + "-key.pem")
	if err != nil {
		return nil, err
	}
	return parseToCert(crt, key)

}

func parseToCert(crt, key []byte) (*tls.Certificate, error) {
	cert, err := tls.X509KeyPair(crt, key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse x509 key pair: %w", err)
	}
	if len(cert.Certificate) == 0 {
		return nil, fmt.Errorf("list of certificates is empty")
	}

	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, err
	}
	return &cert, nil
}
