package cmd

import (
	"github.com/j-martin/bub/core"
	"github.com/urfave/cli"
	"log"
	"os"
)

func BuildCmds() []cli.Command {
	cfg, err := core.LoadConfiguration()
	if err != nil {
		log.Printf("The configuration failed to load... %v", err)
		core.MustSetupConfig()
		log.Print("Run 'bub setup' to complete the setup.")
		os.Exit(0)
	}

	manifest, _ := core.LoadManifest()

	return []cli.Command{
		buildSetupCmd(),
		buildUpdateCmd(cfg),
		buildConfigCmd(cfg),
		{
			Name:        "repository",
			Usage:       "Repository related commands.",
			Aliases:     []string{"r"},
			Subcommands: buildRepositoryCmds(cfg, manifest),
		},
		{
			Name:        "manifest",
			Aliases:     []string{"m"},
			Usage:       "Manifest related commands.",
			Subcommands: buildManifestCmds(cfg),
		},
		buildEC2Cmd(cfg, manifest),
		buildRDSCmd(cfg),
		buildR53Cmd(),
		buildEBCmd(cfg, manifest),
		{
			Name:        "github",
			Usage:       "GitHub related commands.",
			Aliases:     []string{"gh"},
			Subcommands: buildGitHubCmds(cfg, manifest),
		},
		{
			Name:        "jira",
			Usage:       "JIRA related commands.",
			Aliases:     []string{"ji"},
			Subcommands: buildJIRACmds(cfg),
		},
		{
			Name:        "workflow",
			Usage:       "Git/GitHub/JIRA workflow commands.",
			Aliases:     []string{"w"},
			Subcommands: buildWorkflowCmds(cfg, manifest),
		},
		{
			Name:        "jenkins",
			Usage:       "Jenkins related commands.",
			Aliases:     []string{"j"},
			Subcommands: buildJenkinsCmds(cfg, manifest),
		},
		{
			Name:        "splunk",
			Usage:       "Splunk related commands.",
			Aliases:     []string{"s"},
			Subcommands: buildSplunkCmds(cfg, manifest),
		},
		{
			Name:        "confluence",
			Usage:       "Confluence related commands.",
			Aliases:     []string{"c"},
			Subcommands: buildConfluenceCmds(cfg),
		},
		{
			Name:        "circle",
			Usage:       "CircleCI related commands.",
			Subcommands: buildCircleCmds(cfg, manifest),
		},
	}
}
