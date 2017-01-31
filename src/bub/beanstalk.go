package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
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

type Versions []*elasticbeanstalk.ApplicationVersionDescription

func (e Versions) Len() int {
	return len(e)
}

func (e Versions) Less(i, j int) bool {
	date := *e[i].DateUpdated
	return date.After(*e[j].DateUpdated)
}

func (e Versions) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func getBeanstalkSvc() *elasticbeanstalk.ElasticBeanstalk {
	sess, err := session.NewSession(&config)
	if err != nil {
		log.Fatal("Failed to create session,", err)
	}
	return elasticbeanstalk.New(sess)
}

func EnvironmentIsReady(environment string) {
	svc := getBeanstalkSvc()
	lastEvent := time.Now().In(time.UTC)
	previousStatus := ""

	status := elasticbeanstalk.EnvironmentHealthAttributeStatus
	healthStatus := elasticbeanstalk.EnvironmentHealthAttributeHealthStatus
	color := elasticbeanstalk.EnvironmentHealthAttributeColor
	causes := elasticbeanstalk.EnvironmentHealthAttributeCauses

	request := elasticbeanstalk.DescribeEnvironmentHealthInput{
		AttributeNames:  []*string{&status, &healthStatus, &color, &causes},
		EnvironmentName: &environment,
	}

WaitForReady:
	for {
		resp, err := svc.DescribeEnvironmentHealth(&request)
		if err != nil {
			log.Fatal(err.Error())
		}
		if *resp.Status != previousStatus {
			causes := []string{}
			for _, cause := range resp.Causes {
				causes = append(causes, *cause)
			}
			log.Printf("Status: %v, HealthStatus: %v, Color: %v, Causes: %v\n", *resp.Status, *resp.HealthStatus, *resp.Color, strings.Join(causes, ", "))
		}
		previousStatus = *resp.Status
		if *resp.Status == elasticbeanstalk.EnvironmentStatusReady && *resp.HealthStatus == elasticbeanstalk.EnvironmentHealthStatusOk {
			break WaitForReady
		}
		lastEvent = ListEvents(environment, lastEvent, true, false)
		time.Sleep(15 * time.Second)
	}
	log.Println("Done")
}

func DeployVersion(environment string, version string) {
	params := &elasticbeanstalk.UpdateEnvironmentInput{EnvironmentName: &environment, VersionLabel: &version}
	resp, err := getBeanstalkSvc().UpdateEnvironment(params)

	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("Environment: %v, Status: %v", *resp.EnvironmentName, *resp.Status)
	EnvironmentIsReady(environment)
	log.Print("Done")
}

func getEnvironmentVersion() map[string][]string {
	result := make(map[string][]string)
	envResp, err := getBeanstalkSvc().DescribeEnvironments(&elasticbeanstalk.DescribeEnvironmentsInput{})
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, e := range envResp.Environments {
		result[*e.VersionLabel] = append(result[*e.VersionLabel], *e.EnvironmentName)

	}
	return result
}

func ListApplicationVersions(application string) {
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	params := &elasticbeanstalk.DescribeApplicationVersionsInput{}
	if application != "" {
		params.ApplicationName = &application
	}
	resp, err := getBeanstalkSvc().DescribeApplicationVersions(params)
	if err != nil {
		log.Fatal(err.Error())
	}

	versionMapping := getEnvironmentVersion()

	var versions Versions
	versions = resp.ApplicationVersions
	sort.Sort(versions)
	fmt.Fprintln(table, "Application\tVersion\tEnvironment(s)")
	for _, v := range versions {
		row := []string{*v.ApplicationName, *v.VersionLabel, strings.Join(versionMapping[*v.VersionLabel], ", ")}
		fmt.Fprintln(table, strings.Join(row, "\t"))
	}
	table.Flush()
}

func ListEnvironments() {
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	params := &elasticbeanstalk.DescribeEnvironmentsInput{}
	resp, err := getBeanstalkSvc().DescribeEnvironments(params)

	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Fprintln(table, "Application\tEnvironment\tStatus\tHealth\tHealthStatus\tVersionLabel\tCNAME")
	var environments Environments
	environments = resp.Environments
	sort.Sort(environments)
	for _, e := range environments {
		row := []*string{e.ApplicationName, e.EnvironmentName, e.Status, e.Health, e.HealthStatus, e.VersionLabel, e.CNAME}
		fmt.Fprintln(table, joinStringPointers(row, "\t"))
	}
	table.Flush()
}

func ListEvents(environment string, startTime time.Time, reverse bool, header bool) time.Time {
	params := &elasticbeanstalk.DescribeEventsInput{StartTime: &startTime}
	if environment != "" {
		params.EnvironmentName = &environment
	}

	resp, err := getBeanstalkSvc().DescribeEvents(params)

	if err != nil {
		log.Fatal(err.Error())
	}

	table := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	if header {
		fmt.Fprintln(table, "EventDate\tSev.\tEnvironmentName\tMessage")
	}
	var events Events
	events = resp.Events

	lastEvent := time.Time{}
	if len(events) > 0 {
		if reverse {
			sort.Sort(sort.Reverse(events))
			lastEvent = *events[len(events)-1].EventDate
		} else {
			sort.Sort(events)
			lastEvent = *events[0].EventDate
		}
	} else {
		lastEvent = startTime
	}

	for _, e := range events {
		eventDate := *e.EventDate
		var message = *e.Message
		const limit = 200
		if len(message) < limit {
			message = message[0 : len(message)-1]
		} else {
			message = message[0:limit] + "..."
		}

		// EnvironmentName may be nil pointer.
		name := *e.ApplicationName
		if e.EnvironmentName != nil {
			name = *e.EnvironmentName
		}

		if e.EventDate.After(startTime) {
			row := []string{eventDate.Format("2006-01-02 15:04:05Z"), *e.Severity, name, message}
			fmt.Fprintln(table, strings.Join(row, "\t"))
		}
	}
	table.Flush()
	return lastEvent
}
