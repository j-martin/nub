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
)

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
	table.Flush()
	return instances
}

func listInstances(instances []*ec2.Instance) {
	fmt.Fprintln(table, "#\tName\tId\tPublicDNS\tType")
	for c, i := range instances {
		instances = append(instances, i)
		var name *string
		for _, t := range i.Tags {
			if *t.Key == "Name" {
				name = t.Value
			}
		}
		idx := strconv.FormatInt(int64(c), 10)

		row := []string{idx, *name, *i.InstanceId, *i.PublicDnsName, *i.InstanceType}
		fmt.Fprintln(table, strings.Join(row, "\t"))
	}
	table.Flush()
}

func connect(i *ec2.Instance) {
	log.Println(*i)
	usr, _ := user.Current()
	for _, sshUser := range []string{"ubuntu", "ec2-user"} {
		host := sshUser + "@" + *i.PublicDnsName
		key := path.Join(usr.HomeDir, ".ssh", *i.KeyName + ".pem")

		cmd := exec.Command("ssh", "-i", key, host, "-o", "ConnectTimeout=5")
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		log.Printf("Connecting to %v\n", host)

		err := cmd.Run()
		if err == nil {
			break
		}
	}
}

func ConnectToInstance(filter string) {
	instances := FetchInstances(filter)
	if len(instances) == 0 {
		log.Fatal("No instances found.")
	} else if len(instances) == 1 {
		connect(instances[0])
	} else {
		listInstances(instances)
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("Enter a valid instance number: ")
			result, err := reader.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			i, err := strconv.Atoi(strings.Trim(result, "\n"))
			if err == nil && len(instances) > i {
				connect(instances[i])
				break
			}
		}
	}
}
