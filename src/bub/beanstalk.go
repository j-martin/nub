package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"log"
	"os"
	"strings"
	"text/tabwriter"
)

var padding = 2
var w = tabwriter.NewWriter(os.Stdout, 0, 0, padding, ' ', 0)

func ListEnvironments() {
	sess, err := session.NewSession(&config)
	if err != nil {
		log.Fatal("failed to create session,", err)
	}

	svc := elasticbeanstalk.New(sess)

	params := &elasticbeanstalk.DescribeEnvironmentsInput{}
	resp, err := svc.DescribeEnvironments(params)

	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Fprintln(w, "ApplicationName\tEnvironmentName\tStatus\tHealth\tHealthStatus\tVersionLabel\tCNAME")
	for _, e := range resp.Environments {
		row := []string{*e.ApplicationName, *e.EnvironmentName, *e.Status, *e.Health, *e.HealthStatus, *e.VersionLabel, *e.CNAME}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

func ListEvents() {
	sess, err := session.NewSession(&config)
	if err != nil {
		log.Fatal("failed to create session,", err)
	}

	svc := elasticbeanstalk.New(sess)

	params := &elasticbeanstalk.DescribeEventsInput{}
	resp, err := svc.DescribeEvents(params)

	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Fprintln(w, "EventDate\tSev.\tEnvironmentName\tMessage")
	for _, e := range resp.Events {
		time := *e.EventDate
		var message = *e.Message
		const limit = 200
		if len(message) < limit {
			message = message[0 : len(message)-1]
		} else {
			message = message[0:limit] + "..."
		}
		row := []string{time.Format("2006-01-02T15:04:05Z"), *e.Severity, *e.EnvironmentName, message}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}
