package main

import (
	"github.com/aws/aws-sdk-go/service/rds"
	"log"
	"github.com/aws/aws-sdk-go/aws/session"
	"sort"
	"text/tabwriter"
	"os"
	"strings"
	"fmt"
)

func ListRDSInstances(cfg Configuration) {
	table := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "Name\tEndpoint\tRegion\tEngine")

	channel := make(chan []string)
	regions := cfg.Aws.Regions
	for _, region := range regions {
		go func(region string) {
			config := getAwsConfig(region)
			svc := rds.New(session.New(&config))
			resp, err := svc.DescribeDBInstances(&rds.DescribeDBInstancesInput{})
			if err != nil {
				log.Fatal(err)
			}
			var rows []string
			for _, i := range resp.DBInstances {
				name := ""
				if i.DBName != nil {
					name = *i.DBName
				}
				row := []string{name, *i.Endpoint.Address, region, *i.Engine}
				rows = append(rows, strings.Join(row, "\t"))
			}
			channel <- rows
		}(region)
	}

	var instances []string
	for i := 0; i < len(regions); i++ {
		instances = append(instances, <-channel...)
	}
	close(channel)
	sort.Strings(instances)
	for _, i := range instances {
		fmt.Fprintln(table, i)
	}
	table.Flush()
}
