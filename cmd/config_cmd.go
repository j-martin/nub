package cmd

import (
	"github.com/j-martin/bub/core"
	"github.com/urfave/cli"
	"log"
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
			if c.Bool(preview) {
				return core.ShowConfig(cfg)
			}
			log.Printf("Use 'bub config --shared' to edit the shared config.")
			return core.EditConfiguration(core.ConfigUserFile)
		},
	}
}
