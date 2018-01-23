package vault

import (
	"encoding/json"
	"errors"
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/integrations/ssh"
	"github.com/benchlabs/bub/utils"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
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
	err := v.loadToken()
	if err != nil {
		log.Printf("Authentication Failure. Run 'BUB_UPDATE_CREDENTIALS=1 %v' to update your credentials.", strings.Join(os.Args, " "))
		log.Fatalf("Error: %v", err)
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

func (v *Vault) getTokenPath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", nil
	}

	configPath := path.Join(usr.HomeDir, ".config", "bub", "token-"+strings.Replace(v.tunnel.Server.Host, ".", "-", -1))
	return configPath, nil
}

func (v *Vault) loadToken() error {
	filePath, err := v.getTokenPath()
	if err != nil {
		return err
	}
	exists, err := utils.PathExists(filePath)
	if err != nil {
		return err
	}
	if !exists {
		return v.setTokenFromAuth()
	}
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	v.token = string(content)
	return nil
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
	tokenPath, err := v.getTokenPath()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(tokenPath, []byte(token), 0600)
}

type Secret struct {
	RequestID     string            `json:"request_id"`
	LeaseID       string            `json:"lease_id"`
	LeaseDuration int               `json:"lease_duration"`
	Renewable     bool              `json:"renewable"`
	Data          map[string]string `json:"data"`
	Warnings      interface{}       `json:"warnings"`
}

func (v *Vault) read(path string, retry int) (*Secret, error) {
	reader, err := v.tunnel.CommandWithOutput(
		"VAULT_TOKEN="+v.token,
		"vault",
		"read",
		"-format=json",
		path,
	)

	// Super Naive re-auth...
	if err != nil && err.Error() == "Process exited with status 1" && retry > 0 {
		log.Print("Trying to renew token...")
		v.setTokenFromAuth()
		return v.read(path, retry-1)
	} else if err != nil {
		return nil, err
	}
	secret := new(Secret)
	return secret, json.NewDecoder(reader).Decode(secret)
}

func (v *Vault) Read(path string) (*Secret, error) {
	return v.read(path, 1)
}
