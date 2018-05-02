package main

import (
	"github.com/j-martin/bub/cmd"
	"github.com/urfave/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "bub"
	app.Usage = "A tool for all your Bench related needs."
	app.Version = "1.0.0"
	app.EnableBashCompletion = true
	app.Commands = cmd.BuildCmds()
	app.Run(os.Args)
}
