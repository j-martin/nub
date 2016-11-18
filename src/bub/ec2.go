package main

import (
	"bufio"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

type ConnectionParams struct {
	Filter string
	Output bool
	All    bool
	Args   []string
}

func FetchInstances(filter string) []*ec2.Instance {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session %v\n", err)
	}

	svc := ec2.New(sess, &config)
	log.Printf("Fetching instances with tag '%v'\n", filter)
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
		log.Fatalf("there was an error listing instances: %v", err.Error())
	}
	var instances []*ec2.Instance
	for _, r := range resp.Reservations {
		for _, i := range r.Instances {
			instances = append(instances, i)
		}
	}
	return instances
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

func listInstances(instances []*ec2.Instance) {
	table := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "#\tName\tId\tPublicName\tPrivateName\tType")
	for c, i := range instances {
		instances = append(instances, i)

		idx := strconv.FormatInt(int64(c), 10)

		row := []string{idx, getInstanceName(i), *i.InstanceId, *i.PublicDnsName, *i.PrivateDnsName, *i.InstanceType}
		fmt.Fprintln(table, strings.Join(row, "\t"))
	}
	table.Flush()
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

func connect(i *ec2.Instance, params ConnectionParams) {
	if !(params.Output || params.All ) {
		log.Println(*i)
	}
	usr, _ := user.Current()
	for _, sshUser := range getUsers(i) {
		host := sshUser + "@" + *i.PublicDnsName
		key := path.Join(usr.HomeDir, ".ssh", *i.KeyName + ".pem")
		baseArgs := []string{"-i", key, host, "-o", "ConnectTimeout=3"}
		args := append(baseArgs, params.Args...)

		cmd := exec.Command("ssh", args...)
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr

		log.Printf("Connecting -i %v %v\n", key, host)

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
	if len(cmd) > 1 {
		baseArgs := []string{"-tC"}
		switch cmd[1] {
		case "bash":
			cmd = append(append(baseArgs, "/opt/bench/exec bash"), cmd[1:]...)
		case "exec":
			cmd = append(append(baseArgs, "/opt/bench/exec"), cmd[1:]...)
		case "jstack":
			cmd = append(append(baseArgs, "/opt/bench/jstack"), cmd[1:]...)
		case "jmap":
			cmd = append(append(baseArgs, "/opt/bench/jmap"), cmd[1:]...)
		default:
			cmd = append(baseArgs, cmd...)
		}
	}
	return cmd
}

func ConnectToInstance(params ConnectionParams) {
	params.Args = prepareArgs(params)
	instances := FetchInstances(params.Filter)
	if len(instances) == 0 {
		log.Fatal("No instances found.")
	} else if len(instances) == 1 {
		connect(instances[0], params)
	} else if params.Output || params.All {
		for _, i := range instances {
			connect(i, params)
		}
	} else {
		listInstances(instances)
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Fprint(os.Stderr, "Enter a valid instance number: ")
			result, err := reader.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			i, err := strconv.Atoi(strings.Trim(result, "\n"))
			if err == nil && len(instances) > i {
				connect(instances[i], params)
				break
			}
		}
	}
}
