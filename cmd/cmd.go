package cmd

import (
	"log"
	"os"

	"github.com/j-martin/nub/core"
	"github.com/urfave/cli"
)

func BuildCmds() []cli.Command {
	cfg, err := core.LoadConfiguration()
	if err != nil {
		log.Printf("The configuration failed to load... %v", err)
		core.MustSetupConfig()
		log.Print("Run 'nub setup' to complete the setup.")
		os.Exit(0)
	}

	manifest, _ := core.LoadManifest()

	return []cli.Command{
		buildSetupCmd(),
		buildConfigCmd(cfg),
		{
			Name:        "repository",
			Usage:       "Repository related commands.",
			Aliases:     []string{"r"},
			Subcommands: buildRepositoryCmds(cfg, manifest),
		},
		{
			Name:        "github",
			Usage:       "GitHub related commands.",
			Aliases:     []string{"g"},
			Subcommands: buildGitHubCmds(cfg, manifest),
		},
		{
			Name:        "jira",
			Usage:       "JIRA related commands.",
			Aliases:     []string{"j"},
			Subcommands: buildJIRACmds(cfg),
		},
		{
			Name:        "workflow",
			Usage:       "Git/GitHub/JIRA workflow commands.",
			Aliases:     []string{"w"},
			Subcommands: buildWorkflowCmds(cfg, manifest),
		},
		{
			Name:        "confluence",
			Usage:       "Confluence related commands.",
			Aliases:     []string{"c"},
			Subcommands: buildConfluenceCmds(cfg),
		},
	}
}
