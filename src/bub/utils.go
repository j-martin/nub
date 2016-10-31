package main

import (
	"bufio"
	"os"
	"fmt"
	"log"
	"strings"
	"text/tabwriter"
)

var table = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}
