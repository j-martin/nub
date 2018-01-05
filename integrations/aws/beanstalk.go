package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elasticbeanstalk"
	"github.com/benchlabs/bub/core"
	"github.com/benchlabs/bub/utils"
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

func getBeanstalkSvc(region string) *elasticbeanstalk.ElasticBeanstalk {
	config := GetAWSConfig(region)
	sess, err := session.NewSession(&config)
	if err != nil {
		log.Fatal("Failed to create session,", err)
	}
	return elasticbeanstalk.New(sess)
}

func GetApplication(environment string) string {
	result := strings.Split(environment, "-")
	return strings.Join(result[1:], "-")
}

func EnvironmentIsReady(region string, environment string, failOnError bool) {
	svc := getBeanstalkSvc(region)
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

	for {
		resp, err := svc.DescribeEnvironmentHealth(&request)
		if err != nil {
			log.Fatal(err.Error())
		}
		if *resp.Status != previousStatus {
			var causes []string
			for _, cause := range resp.Causes {
				causes = append(causes, *cause)
			}
			log.Printf("status: %v, healthstatus: %v, color: %v, causes: %v\n", *resp.Status, *resp.HealthStatus, *resp.Color, strings.Join(causes, ", "))
		}
		previousStatus = *resp.Status
		if *resp.Status == elasticbeanstalk.EnvironmentStatusReady && *resp.HealthStatus == elasticbeanstalk.EnvironmentHealthStatusOk {
			break
		}

		lastEvent = ListEvents(region, environment, lastEvent, true, false, failOnError)
		time.Sleep(30 * time.Second)
	}
	log.Println("Done")
}

func versionAlreadyDeployed(svc *elasticbeanstalk.ElasticBeanstalk, region string, environment string, version string) {
	params := elasticbeanstalk.DescribeEnvironmentsInput{EnvironmentNames: []*string{&environment}}
	environments, err := svc.DescribeEnvironments(&params)
	if err != nil {
		log.Fatal(err.Error())
	}
	if len(environments.Environments) == 0 {
		log.Fatalf("No environment found for %v in %v", environment, region)
	}
	currentVersion := *environments.Environments[0].VersionLabel
	if currentVersion == version {
		log.Print("The same version is already deployed. skipping.")
		os.Exit(0)
	}
	log.Printf("Updating from verson %s to %s", currentVersion, version)
}

func DescribeEnvironment(region string, environment string, all bool) {
	application := strings.Split(environment, "-")[0]
	params := &elasticbeanstalk.DescribeConfigurationSettingsInput{ApplicationName: &application, EnvironmentName: &environment}

	svc := getBeanstalkSvc(region)
	resp, err := svc.DescribeConfigurationSettings(params)
	if err != nil {
		log.Fatal(err.Error())
	}
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "Option\tValue")
	for _, s := range resp.ConfigurationSettings {
		for _, o := range s.OptionSettings {
			if o.Value != nil {
				if all || *o.Namespace == "aws:elasticbeanstalk:application:environment" {
					fmt.Fprintln(table, strings.Join([]string{*o.OptionName, *o.Value}, "\t"))
				}
			}
		}
	}
	table.Flush()
}

func DeployVersion(region string, environment string, version string) {
	svc := getBeanstalkSvc(region)
	params := &elasticbeanstalk.DescribeEnvironmentsInput{EnvironmentNames: []*string{&environment}}
	retries := 50
	for {
		resp, err := svc.DescribeEnvironments(params)
		if err != nil {
			log.Fatalf("Could not describe the environment: %v", err)
		}

		if len(resp.Environments) != 1 {
			log.Fatal("None or more than one environment was found. cannot continue.")
		}

		description := resp.Environments[0]
		if *description.Status == elasticbeanstalk.EnvironmentStatusReady {
			break
		}

		retries -= 1
		if retries < 0 {
			log.Fatal("No more retries left.")
		}

		log.Print("Waiting for the environment to ready.")
		time.Sleep(30 * time.Second)
	}

	versionAlreadyDeployed(svc, region, environment, version)
	updateParams := &elasticbeanstalk.UpdateEnvironmentInput{EnvironmentName: &environment, VersionLabel: &version}
	resp, err := svc.UpdateEnvironment(updateParams)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("Environment: %v, Status: %v", *resp.EnvironmentName, *resp.Status)
	EnvironmentIsReady(region, environment, true)
	log.Print("Done")
}

func getEnvironmentVersion(region string) map[string][]string {
	result := make(map[string][]string)
	envResp, err := getBeanstalkSvc(region).DescribeEnvironments(&elasticbeanstalk.DescribeEnvironmentsInput{})
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, e := range envResp.Environments {
		result[*e.VersionLabel] = append(result[*e.VersionLabel], *e.EnvironmentName)

	}
	return result
}

func ListApplicationVersions(region string, application string) {
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	params := &elasticbeanstalk.DescribeApplicationVersionsInput{}
	if application != "" {
		params.ApplicationName = &application
	}
	resp, err := getBeanstalkSvc(region).DescribeApplicationVersions(params)
	if err != nil {
		log.Fatal(err.Error())
	}

	versionMapping := getEnvironmentVersion(region)

	var versions Versions
	versions = resp.ApplicationVersions
	sort.Sort(versions)
	fmt.Fprintln(table, "Application\tVersion\tRegion\tEnvironment(s)")
	for _, v := range versions {
		row := []string{*v.ApplicationName, *v.VersionLabel, region, strings.Join(versionMapping[*v.VersionLabel], ", ")}
		fmt.Fprintln(table, strings.Join(row, "\t"))
	}
	table.Flush()
}

func ListEnvironments(cfg *core.Configuration) {
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "Application\tEnvironment\tRegion\tStatus\tHealth\tHealthStatus\tVersionLabel\tCNAME")

	params := &elasticbeanstalk.DescribeEnvironmentsInput{}
	channel := make(chan []string)
	for _, region := range cfg.AWS.Regions {
		go func(region string) {
			log.Printf("Listing environments in %v...", region)
			resp, err := getBeanstalkSvc(region).DescribeEnvironments(params)
			if err != nil {
				log.Fatal(err.Error())
			}
			var rows []string
			for _, e := range resp.Environments {
				row := []*string{e.ApplicationName, e.EnvironmentName, &region, e.Status, e.Health, e.HealthStatus, e.VersionLabel, e.CNAME}
				rows = append(rows, utils.JoinStringPointers(row, "\t"))
			}
			channel <- rows
		}(region)
	}

	var environments []string
	for i := 0; i < len(cfg.AWS.Regions); i++ {
		environments = append(environments, <-channel...)
	}
	close(channel)
	sort.Strings(environments)

	for _, e := range environments {
		fmt.Fprintln(table, e)
	}
	table.Flush()
}

func ListEvents(region string, environment string, startTime time.Time, reverse bool, header bool, failOnError bool) time.Time {
	params := &elasticbeanstalk.DescribeEventsInput{StartTime: &startTime}
	if environment != "" {
		params.EnvironmentName = &environment
	}

	resp, err := getBeanstalkSvc(region).DescribeEvents(params)

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

		if failOnError && *e.Severity == elasticbeanstalk.EventSeverityError {
			table.Flush()
			log.Fatal("There was an error in the deployment.")
		}
	}
	table.Flush()
	return lastEvent
}
