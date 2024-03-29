package vault

import (
	"github.com/fraima/key-keeper/internal/config"
	"github.com/fraima/key-keeper/internal/controller"
)

type Client interface {
	Read(path string) (map[string]interface{}, error)
	Write(path string, data map[string]interface{}) (map[string]interface{}, error)
	Put(kvMountPath, secretePath string, data map[string]interface{}) error
	Get(kvMountPath, secretePath string) (map[string]interface{}, error)
}

type vault struct {
	cli Client

	name        string
	role        string
	caPath      string
	rootCAPath  string
	kv          string
	certificate map[string]config.Certificate
}

func Connector(
	connect func(name string, cfg config.Vault) (Client, error),
) func(cfg config.Issuer) (controller.Issuer, error) {
	return func(cfg config.Issuer) (controller.Issuer, error) {
		driver, err := connect(cfg.Name, cfg.Vault)
		if err != nil {
			return nil, err
		}

		v := &vault{
			cli:         driver,
			name:        cfg.Name,
			role:        cfg.Vault.Resource.Role,
			caPath:      cfg.Vault.Resource.CAPath,
			rootCAPath:  cfg.Vault.Resource.RootCAPath,
			kv:          cfg.Vault.Resource.KV.Path,
			certificate: make(map[string]config.Certificate),
		}
		return v, nil
	}
}

func (s *vault) Name() string {
	return s.name
}

func (s *vault) AddResource(r config.Resources) {
	for _, cert := range r.Certificates {
		s.certificate[cert.Name] = cert
	}
	for _, secret := range r.Secrets {
		go func(secret config.Secret) {
			s.ensureSecret(secret)
		}(secret)
	}
	s.EnsureResource()
}

func (s *vault) EnsureResource() {
	for _, cert := range s.certificate {
		go func(c config.Certificate) {
			if c.IsCA {
				s.ensureCA(c)
				return
			}
			s.ensureCertificate(c)
		}(cert)
	}
}
