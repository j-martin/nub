package cmd

import (
	"errors"
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/integrations/vault"
	"github.com/benchlabs/bub/utils"
	"github.com/benchlabs/bub/utils/ssh"
	"github.com/urfave/cli"
	"io/ioutil"
	"log"
	"strings"
)

func buildConfigCmd(cfg *core.Configuration) cli.Command {
	showDefaults := "show-default"
	shared := "shared"
	loadSharedConfigOpt := "load-shared-config"
	storeSharedConfigOpt := "store-shared-config"
	preview := "preview"
	return cli.Command{
		Name:  "config",
		Usage: "Edit your bub config.",
		Flags: []cli.Flag{
			cli.BoolFlag{Name: showDefaults, Usage: "Show default config for reference"},
			cli.BoolFlag{Name: shared, Usage: "Edit shared config."},
			cli.BoolFlag{Name: loadSharedConfigOpt, Usage: "Load shared config to your config."},
			cli.BoolFlag{Name: storeSharedConfigOpt, Usage: "Store shared config."},
			cli.BoolFlag{Name: preview, Usage: "Show/preview the final config."},
		},
		Action: func(c *cli.Context) error {
			if c.Bool(showDefaults) {
				print(core.GetConfigString())
				return nil
			}
			if c.Bool(shared) {
				return core.EditConfiguration(core.ConfigSharedFile)
			}
			if c.Bool(loadSharedConfigOpt) {
				return loadSharedConfig(cfg)
			}
			if c.Bool(storeSharedConfigOpt) {
				return storeSharedConfig(cfg)
			}
			if c.Bool(preview) {
				return core.ShowConfig(cfg)
			}
			log.Printf("Use 'bub config --shared' to edit the shared config.")
			return core.EditConfiguration(core.ConfigUserFile)
		},
	}
}

func GetEnvironment(cfg *core.Configuration, env string) (*core.Environment, error) {
	for _, e := range cfg.AWS.Environments {
		if strings.HasPrefix(env, e.Prefix) {
			return &e, nil
		}
	}
	return nil, errors.New("no environment found")
}

func prepareTunnel(cfg *core.Configuration, env string) (*ssh.Connection, error) {
	environment, err := GetEnvironment(cfg, env)
	if err != nil {
		return nil, err
	}
	tunnel := ssh.Connection{
		JumpHost: environment.JumpHost,
		Tunnels: map[string]ssh.Tunnel{
			"vault": vault.GetVaultTunnelConfiguration(environment),
		},
	}
	tunnel.Connect()
	return &tunnel, nil
}

func storeSharedConfig(cfg *core.Configuration) error {
	if !utils.AskForConfirmation("Store the shared config to Vault?") {
		log.Print("Aborting...")
		return nil
	}
	tunnel, err := prepareTunnel(cfg, "dev")
	if err != nil {
		return err
	}
	data, err := ioutil.ReadFile(core.GetConfigPath(core.ConfigSharedFile))
	if err != nil {
		return err
	}
	payload := make(map[string]interface{})
	payload["shared"] = string(data)

	_, err = vault.MustInitVault(cfg, tunnel).Write(cfg.Vault.Path, payload)
	return err
}

func loadSharedConfig(cfg *core.Configuration) error {
	tunnel, err := prepareTunnel(cfg, "dev")
	if err != nil {
		return err
	}
	secret, err := vault.MustInitVault(cfg, tunnel).Read(cfg.Vault.Path)
	if err != nil {
		return err
	}
	if data, ok := secret.Data["shared"]; ok {
		configPath := core.GetConfigPath(core.ConfigSharedFile)
		exists, err := utils.PathExists(configPath)
		if err != nil {
			return err
		}
		if exists {
			err = utils.Copy(configPath, configPath+".bak")
			if err != nil {
				return err
			}
		}
		return ioutil.WriteFile(
			configPath,
			[]byte(data.(string)),
			0600,
		)
	} else {
		return errors.New("no shared config found")
	}
}
