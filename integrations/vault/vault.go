package vault

import (
	"errors"
	"fmt"
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/utils"
	"github.com/benchlabs/bub/utils/ssh"
	"github.com/hashicorp/vault/api"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"strings"
)

type Vault struct {
	tokenName string
	cfg       *core.Configuration
	client    *api.Client
}

func GetVaultTunnelConfiguration(env *core.Environment) ssh.Tunnel {
	return ssh.Tunnel{RemoteHost: "vault." + env.Domain, LocalPort: ssh.GetPort(), RemotePort: 8200}
}

func MustInitVault(cfg *core.Configuration, s ssh.Connection) *Vault {
	mustLoadVaultCredentials(cfg)
	tunnel := s.Tunnels["vault"]
	v := &Vault{cfg: cfg, tokenName: "token." + tunnel.RemoteHost}
	vaultCfg := api.DefaultConfig()
	vaultCfg.Address = fmt.Sprintf("%v:%v", cfg.Vault.Server, tunnel.LocalPort)
	client, err := api.NewClient(vaultCfg)
	if err != nil {
		log.Fatalf("Failed to get Vault client: %v", err)
	}
	v.client = client
	err = v.loadToken()
	if err != nil {
		log.Fatalf("Failed to load token: %v", err)
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
	configPath := path.Join(usr.HomeDir, ".config", "bub", v.tokenName)
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
	v.client.SetToken(strings.Trim(string(content), "\n"))
	return nil
}

func (v *Vault) setTokenFromAuth() error {
	urlPath := strings.ToLower(fmt.Sprintf("auth/%s/login/%s", v.cfg.Vault.AuthMethod, v.cfg.Vault.Username))
	data := map[string]interface{}{
		"password": v.cfg.Vault.Password,
	}
	secret, err := v.client.Logical().Write(urlPath, data)
	if err != nil {
		log.Fatal(err)
	}
	token, err := secret.TokenID()
	if err != nil {
		log.Print("Authentication Failure.")
		log.Printf("Run 'BUB_UPDATE_CREDENTIALS=1 %v' to change your credentials.", strings.Join(os.Args, " "))
		log.Fatalf("Error: %v", err)
	}
	if token == "" {
		return errors.New("failed to get authenticated and get Vault token")
	}
	v.client.SetToken(token)
	tokenPath, err := v.getTokenPath()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(tokenPath, []byte(token), 0600)
}

func (v *Vault) read(path string, retries int) (*api.Secret, error) {
	secret, err := v.client.Logical().Read(path)
	if err != nil && retries >= 0 {
		_, err := v.client.Auth().Token().LookupSelf()
		if strings.Contains(err.Error(), "Code: 403.") {
			log.Print("Trying to renew token...")
			v.setTokenFromAuth()
		} else if err != nil {
			return secret, err
		}
		return v.read(path, retries-1)
	}
	return secret, err
}

func (v *Vault) Read(path string) (*api.Secret, error) {
	return v.read(path, 2)
}
