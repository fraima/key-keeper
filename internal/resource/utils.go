package resource

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"

	"github.com/fraima/key-keeper/internal/config"
)

func (s *resource) storeKey(path string, privare, public []byte) error {
	if err := os.WriteFile(path+".pem", privare, 0600); err != nil {
		return fmt.Errorf("failed to save privare key with path %s: %w", path, err)
	}

	if err := os.WriteFile(path+".pub", public, 0600); err != nil {
		return fmt.Errorf("failed to public key file: %w", err)
	}
	return nil
}

func (s *resource) storeKeyPair(path string, crt, key []byte) error {
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

func (s *resource) readCertificate(path string) (*x509.Certificate, error) {
	crt, err := os.ReadFile(path + ".pem")
	if err != nil {
		return nil, err
	}

	pBlock, _ := pem.Decode(crt)
	return x509.ParseCertificate(pBlock.Bytes)
}

func (s *resource) readCA(vaultPath string) (crt, key []byte, err error) {
	vaultPath = path.Join(vaultPath, "cert/ca_chain")
	ica, err := s.vault.Read(vaultPath)
	if ica != nil {
		if c, ok := ica["certificate"]; ok {
			crt = []byte(c.(string))
		}
		if k, ok := ica["private_key"]; ok {
			key = []byte(k.(string))
		}
	}
	return
}

func createCSR(spec config.Spec) (crt, key []byte) {
	pk, _ := rsa.GenerateKey(rand.Reader, spec.PrivateKey.Size)

	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:         spec.Subject.CommonName,
			Country:            spec.Subject.Country,
			Locality:           spec.Subject.Locality,
			Organization:       spec.Subject.Organization,
			OrganizationalUnit: spec.Subject.OrganizationalUnit,
			Province:           spec.Subject.Province,
			PostalCode:         spec.Subject.PostalCode,
			StreetAddress:      spec.Subject.StreetAddress,
			SerialNumber:       spec.Subject.SerialNumber,
		},
		IPAddresses:        getIPAddresses(spec.IPAddresses),
		URIs:               getURIs(spec.Hostnames),
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	csr, _ := x509.CreateCertificateRequest(rand.Reader, &template, pk)

	//pem encoding of certificate
	return pem.EncodeToMemory(
			&pem.Block{
				Type:  "CERTIFICATE REQUEST",
				Bytes: csr,
			},
		), pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(pk),
			},
		)

}

func getIPAddresses(cfg config.IPAddresses) []net.IP {
	ipAddresses := make(map[string]net.IP)

	for _, ip := range cfg.Static {
		ipAddresses[ip] = net.IP(ip)
	}

	ifaces, _ := net.Interfaces()
	// TODO: handle err
	for _, i := range ifaces {
		if inSlice(i.Name, cfg.Interfaces) {
			addrs, _ := i.Addrs()
			// TODO: handle err
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				ipAddresses[ip.String()] = ip
			}
		}
	}

	for _, h := range cfg.DNSLookup {
		ips, _ := net.LookupIP(h)
		for _, ip := range ips {
			ipAddresses[ip.String()] = ip
		}
	}

	r := make([]net.IP, len(ipAddresses))
	for _, ip := range ipAddresses {
		r = append(r, ip)
	}
	return r
}

func getURIs(hostnames []string) []*url.URL {
	urls := make([]*url.URL, 0, len(hostnames))

	for _, hostname := range hostnames {
		// TODO: error handler
		url, _ := url.Parse(hostname)
		urls = append(urls, url)
	}
	return urls
}

func inSlice(str string, sl []string) bool {
	for _, s := range sl {
		if str == s {
			return true
		}
	}
	return false
}