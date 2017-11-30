package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/manifoldco/promptui"
)

type ConnectionParams struct {
	Configuration Configuration
	Filter        string
	Output        bool
	All           bool
	UseJumpHost   bool
	Args          []string
}

func FetchInstances(done chan []*ec2.Instance, region string, filter string) {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session %v\n", err)
	}

	config := getAWSConfig(region)
	svc := ec2.New(sess, &config)
	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String(strings.Join([]string{"*", filter, "*"}, "")),
				},
			},
			{
				Name: aws.String("instance-state-name"),
				Values: []*string{
					aws.String("running"),
				},
			},
		},
	}
	resp, err := svc.DescribeInstances(params)
	if err != nil {
		log.Fatalf("There was an error listing instances: %v", err.Error())
	}
	var instances []*ec2.Instance
	for _, r := range resp.Reservations {
		for _, i := range r.Instances {
			instances = append(instances, i)
		}
	}
	done <- instances
	return
}

func getInstanceName(i *ec2.Instance) string {
	var name string
	for _, t := range i.Tags {
		if *t.Key == "Name" {
			name = *t.Value
		}
	}
	return name
}

func getUsers(i *ec2.Instance) []string {
	users := []string{}
	for _, t := range i.Tags {
		if *t.Key == "elasticbeanstalk:environment-name" {
			users = append(users, "ec2-user")
		}
	}
	return append(users, "ubuntu")
}

func getJumpHost(name string, cfg Configuration) string {
	for _, i := range cfg.AWS.Environments {
		if strings.HasPrefix(name, i.Prefix) {
			return i.Jumphost
		}
	}
	log.Fatal("Could not find jump host in configuration.")
	return ""
}

func connect(i *ec2.Instance, params ConnectionParams) {
	if !(params.Output || params.All) {
		log.Println(*i)
	}
	usr, _ := user.Current()
	hostname := *i.PublicDnsName
	key := path.Join(usr.HomeDir, ".ssh", *i.KeyName+".pem")
	baseArgs := []string{}

	if hostname == "" || params.UseJumpHost {
		hostname = *i.PrivateDnsName
		jumpHost := getJumpHost(getInstanceName(i), params.Configuration)
		log.Printf("No public DNS name found, using jump host: %v", jumpHost)
		baseArgs = []string{"-A", "-J", jumpHost}
	}

	for _, sshUser := range getUsers(i) {
		host := sshUser + "@" + hostname
		connectTimeout := params.Configuration.Ssh.ConnectTimeout
		if connectTimeout == 0 {
			connectTimeout = 3
		}
		args := append(baseArgs, "-i", key, host, "-o", fmt.Sprintf("ConnectTimeout=%d", connectTimeout))
		args = append(args, params.Args...)

		cmd := exec.Command("ssh", args...)
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr

		log.Printf("Connecting %v\n", strings.Join(args, " "))

		var err error
		if params.Output {
			err = saveCommandOutput(i, cmd)
		} else {
			cmd.Stdout = os.Stdout
			err = cmd.Run()
		}
		if err == nil {
			break
		}
	}
}

func saveCommandOutput(i *ec2.Instance, cmd *exec.Cmd) error {
	content, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	outputPath := "output-" + getInstanceName(i) + "-" + time.Now().Format("2006-01-02T15-04-05Z") + ".txt"
	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.Write(content)

	if err != nil {
		log.Fatal(err)
	}
	f.Close()
	log.Printf("Saved output to: %v", outputPath)

	return err
}

func prepareArgs(params ConnectionParams) []string {
	cmd := params.Args
	if len(cmd) > 0 {
		baseArgs := []string{"-tC"}
		switch cmd[0] {
		case "tmux":
			arg := ""
			usr, _ := user.Current()
			if os.Getenv("TERM_PROGRAM") == "iTerm.app" && os.Getenv("TMUX") == "" {
				arg = "-CC"
			}
			tmuxCmd := []string{
				"tmux", arg, "attach", "-t", usr.Username, "||",
				"tmux", arg, "new", "-s", usr.Username,
			}
			cmd = append(append(baseArgs, strings.Join(tmuxCmd, " ")), cmd[1:]...)
		case "bash":
			cmd = append(append(baseArgs, "/opt/bench/exec bash"), cmd[1:]...)
		case "exec":
			cmd = append(append(baseArgs, "/opt/bench/exec"), cmd[1:]...)
		case "jstack":
			cmd = append(append(baseArgs, "/opt/bench/jstack"), cmd[1:]...)
		case "jmap":
			cmd = append(append(baseArgs, "/opt/bench/jmap"), cmd[1:]...)
		case "logs":
			cmd = append(append(baseArgs, "/opt/bench/logs"), cmd[1:]...)
		default:
			cmd = append(baseArgs, cmd...)
		}
	}
	return cmd
}

func ConnectToInstance(params ConnectionParams) {
	params.Args = prepareArgs(params)
	var instances []*ec2.Instance

	channel := make(chan []*ec2.Instance)
	regions := params.Configuration.AWS.Regions
	log.Printf("Fetching instances with tag '%v'", params.Filter)

	for _, region := range regions {
		go FetchInstances(channel, region, params.Filter)
	}
	for i := 0; i < len(regions); i++ {
		instances = append(instances, <-channel...)
	}
	close(channel)

	if len(instances) == 0 {
		log.Fatal("No instances found.")
	} else if len(instances) == 1 {
		connect(instances[0], params)
	} else if params.Output || params.All {
		for _, i := range instances {
			connect(i, params)
		}
	} else {

		// The Architecture field is being overwritten
		// with the instance name tag to make it easier to template.
		// The alternative was to define a new struct.
		templates := &promptui.SelectTemplates{
			Label: "{{ . }}:",
			Active: "> {{ .InstanceId }}	{{ .Architecture }}",
			Inactive: "  {{ .InstanceId }}	{{ .Architecture }}",
			Selected: "= {{ .InstanceId }}	{{ .Architecture }}",
			Details: `
--------- Instance ----------
{{ "Id:" | faint }}	{{ .InstanceId }}
{{ "Name:" | faint }}	{{ .Architecture }}
{{ "LaunchTime:" | faint }}	{{ .LaunchTime }}
{{ "PublicDnsName:" | faint }}	{{ .PublicDnsName }}
{{ "PrivateDnsName:" | faint }}	{{ .PrivateDnsName }}
{{ "InstanceType:" | faint }}	{{ .InstanceType }}
{{ "PublicIpAddress:" | faint }}	{{ .PublicIpAddress }}
{{ "PrivateIpAddress:" | faint }}	{{ .PrivateIpAddress }}
`,
		}

		searcher := func(input string, index int) bool {
			i := instances[index]
			name := strings.Replace(strings.ToLower(*i.Architecture), " ", "", -1)
			input = strings.Replace(strings.ToLower(input), " ", "", -1)

			return strings.Contains(name, input)
		}

		for i := range instances {
			name := getInstanceName(instances[i])
			instances[i].Architecture = &name
		}

		prompt := promptui.Select{
			Size:      20,
			Label:     "Select an EC2 Instance",
			Items:     instances,
			Templates: templates,
			Searcher:  searcher,
		}

		i, _, err := prompt.Run()
		if err != nil {
			log.Fatalf("Failed to pick instance: %v", err)
		}
		connect(instances[i], params)
	}
}
