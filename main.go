package main

import (
	"github.com/benchlabs/bub/cmd"
	"github.com/benchlabs/bub/core"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "bub"
	app.Usage = "A tool for all your Bench related needs."
	app.Version = "0.32.0"
	app.EnableBashCompletion = true
	cmd.Init(app, core.LoadConfiguration())
}
