package main

import (
	"github.com/benchlabs/bub/cmd"
	"github.com/urfave/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "bub"
	app.Usage = "A tool for all your Bench related needs."
	app.Version = "0.57.1"
	app.EnableBashCompletion = true
	app.Commands = cmd.BuildCmds()
	app.Run(os.Args)
}
