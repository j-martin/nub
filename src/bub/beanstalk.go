package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"log"
	"strings"
	"text/tabwriter"
	"os"
)

func getBeanstalkSvc() *elasticbeanstalk.ElasticBeanstalk {
	sess, err := session.NewSession(&config)
	if err != nil {
		log.Fatal("Failed to create session,", err)
	}
	return elasticbeanstalk.New(sess)

}
func ListEnvironments() {
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	params := &elasticbeanstalk.DescribeEnvironmentsInput{}
	resp, err := getBeanstalkSvc().DescribeEnvironments(params)

	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Fprintln(table, "ApplicationName\tEnvironmentName\tStatus\tHealth\tHealthStatus\tVersionLabel\tCNAME")
	for _, e := range resp.Environments {
		row := []string{*e.ApplicationName, *e.EnvironmentName, *e.Status, *e.Health, *e.HealthStatus, *e.VersionLabel, *e.CNAME}
		fmt.Fprintln(table, strings.Join(row, "\t"))
	}
	table.Flush()
}

func ListEvents() {
	params := &elasticbeanstalk.DescribeEventsInput{}
	resp, err := getBeanstalkSvc().DescribeEvents(params)

	if err != nil {
		log.Fatal(err.Error())
	}

	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "EventDate\tSev.\tEnvironmentName\tMessage")
	for _, e := range resp.Events {
		time := *e.EventDate
		var message = *e.Message
		const limit = 200
		if len(message) < limit {
			message = message[0 : len(message) - 1]
		} else {
			message = message[0:limit] + "..."
		}
		row := []string{time.Format("2006-01-02T15:04:05Z"), *e.Severity, *e.EnvironmentName, message}
		fmt.Fprintln(table, strings.Join(row, "\t"))
	}
	table.Flush()
}
