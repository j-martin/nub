package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"log"
	"os"
	"strings"
	"text/tabwriter"
	"sort"
)

type Environments []*elasticbeanstalk.EnvironmentDescription

func (e Environments) Len() int {
	return len(e)
}

func (e Environments) Less(i, j int) bool {
	return *e[i].ApplicationName < *e[j].ApplicationName
}

func (e Environments) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

type Events []*elasticbeanstalk.EventDescription

func (e Events) Len() int {
	return len(e)
}

func (e Events) Less(i, j int) bool {
	date := *e[i].EventDate
	return date.After(*e[j].EventDate)
}

func (e Events) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

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
	var environments Environments
	environments = resp.Environments
	sort.Sort(environments)
	for _, e := range environments {
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
	var events Events
	events = resp.Events
	sort.Sort(events)
	for _, e := range events {
		time := *e.EventDate
		var message = *e.Message
		const limit = 200
		if len(message) < limit {
			message = message[0 : len(message) - 1]
		} else {
			message = message[0:limit] + "..."
		}

		// EnvironmentName may be nil pointer.
		name := *e.ApplicationName
		if e.EnvironmentName != nil {
			name = *e.EnvironmentName
		}
		row := []string{time.Format("2006-01-02T15:04:05Z"), *e.Severity, name, message}
		fmt.Fprintln(table, strings.Join(row, "\t"))
	}
	table.Flush()
}
