package vault

import (
	"context"
	"fmt"
	"path"
	"time"

	auth "github.com/hashicorp/vault/api/auth/approle"
	"go.uber.org/zap"

	"github.com/fraima/key-keeper/internal/config"
)

func (s *vault) auth(name string, a config.Auth) error {
	roleID, err := s.roleID(name, a.AppRole)
	if err != nil {
		return fmt.Errorf("get role id: %w", err)
	}
	secretID, err := s.secretID(name, a.AppRole)
	if err != nil {
		return fmt.Errorf("get secret id: %w", err)
	}

	appRoleAuth, err := auth.NewAppRoleAuth(
		roleID,
		&auth.SecretID{
			FromString: secretID,
		},
		auth.WithMountPath(a.AppRole.Path),
	)
	if err != nil {
		return err
	}

	ttl, err := s.updateAuthToken(appRoleAuth)
	if err != nil {
		return err
	}

	go func() {
		t := time.NewTimer(ttl)
		for range t.C {
			ttl, err := s.updateAuthToken(appRoleAuth)
			if err != nil {
				zap.L().Error("update auth token", zap.Error(err))
			}
			t.Reset(ttl)
		}
	}()
	return nil
}

func (s *vault) roleID(name string, appRole config.AppRole) (string, error) {
	if roleID, rErr := readFromFile(appRole.RoleIDLocalPath); rErr == nil {
		return string(roleID), nil
	}

	vaultPath := path.Join("auth", appRole.Path, "role", appRole.Name, "role-id")
	approle, err := s.Read(vaultPath)
	if err != nil {
		return "", fmt.Errorf("read role_id for path: %s : %w", vaultPath, err)
	}
	if approle == nil {
		return "", fmt.Errorf("no role_id info was returned")
	}

	roleID, ok := approle["role_id"]
	if !ok {
		return "", fmt.Errorf("not found role_id")
	}

	if err = writeToFile(appRole.RoleIDLocalPath, []byte(roleID.(string))); err != nil {
		return "", fmt.Errorf("save role id path: %s : %w", appRole.RoleIDLocalPath, err)
	}
	return roleID.(string), err
}

func (s *vault) secretID(name string, appRole config.AppRole) (string, error) {
	if secretID, rErr := readFromFile(appRole.SecretIDLocalPath); rErr == nil {
		return string(secretID), nil
	}

	vaultPath := path.Join("auth", appRole.Path, "role", appRole.Name, "secret-id")
	approle, err := s.Write(vaultPath, nil)
	if err != nil {
		return "", fmt.Errorf("read secrete_id for path: %s : %w", vaultPath, err)
	}
	if approle == nil {
		return "", fmt.Errorf("no secrete_id info was returned")
	}

	secretID, ok := approle["secret_id"]
	if !ok {
		return "", fmt.Errorf("not found secrete_id")
	}

	if err = writeToFile(appRole.SecretIDLocalPath, []byte(secretID.(string))); err != nil {
		return "", fmt.Errorf("save secret id path: %s : %w", appRole.SecretIDLocalPath, err)
	}
	return secretID.(string), err
}

func (s *vault) updateAuthToken(appRoleAuth *auth.AppRoleAuth) (time.Duration, error) {
	authInfo, err := s.cli.Auth().Login(context.Background(), appRoleAuth)
	if err != nil {
		return 0, err
	}
	if authInfo == nil {
		return 0, fmt.Errorf("no auth info was returned after login")
	}

	token, err := authInfo.TokenID()
	if err != nil {
		return 0, err
	}
	s.cli.SetToken(token)

	ttl, err := authInfo.TokenTTL()
	if err != nil {
		return 0, err
	}
	return ttl, nil
}
