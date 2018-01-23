package vault

import (
	"encoding/json"
	"errors"
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/integrations/ssh"
	"log"
	"strings"
)

type Vault struct {
	cfg    *core.Configuration
	tunnel *ssh.Tunnel
	token  string
}

func MustInitVault(cfg *core.Configuration, t *ssh.Tunnel) *Vault {
	mustLoadVaultCredentials(cfg)
	v := &Vault{cfg: cfg, tunnel: t}
	err := v.setTokenFromAuth()
	if err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}
	return v
}

func MustSetupVault(cfg *core.Configuration) {
	mustLoadVaultCredentials(cfg)
}

func mustLoadVaultCredentials(cfg *core.Configuration) {
	if cfg.Vault.AuthMethod == "" {
		cfg.Vault.AuthMethod = "Okta"
	}
	err := core.LoadCredentials("Vault/"+cfg.Vault.AuthMethod, &cfg.Vault.Username, &cfg.Vault.Password, cfg.ResetCredentials)
	if err != nil {
		log.Fatalf("Failed to set Vault credentials: %v", err)
	}
}

func (v *Vault) setTokenFromAuth() error {
	token, err := v.tunnel.CommandWithStrOutput(
		"vault",
		"auth",
		"-token-only",
		"-method="+strings.ToLower(v.cfg.Vault.AuthMethod),
		"username="+v.cfg.Vault.Username,
		"password="+v.cfg.Vault.Password,
	)
	if err != nil {
		return err
	}
	if token == "" {
		return errors.New("failed to get authenticated and get Vault token")
	}
	v.token = token
	return err
}

type Secret struct {
	RequestID     string            `json:"request_id"`
	LeaseID       string            `json:"lease_id"`
	LeaseDuration int               `json:"lease_duration"`
	Renewable     bool              `json:"renewable"`
	Data          map[string]string `json:"data"`
	Warnings      interface{}       `json:"warnings"`
}

func (v *Vault) Read(path string) (*Secret, error) {
	reader, err := v.tunnel.CommandWithOutput(
		"VAULT_TOKEN="+v.token,
		"vault",
		"read",
		"-format=json",
		path,
	)
	if err != nil {
		return nil, err
	}
	secret := new(Secret)
	return secret, json.NewDecoder(reader).Decode(secret)
}
