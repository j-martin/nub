package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/benchlabs/bub/utils"
	"github.com/manifoldco/promptui"
	"os"
	"strings"
	"sync"
	"text/template"
)

type RecordSet struct {
	Zone      *route53.HostedZone
	RecordSet *route53.ResourceRecordSet
}

func SearchRecords(filter string) (records []RecordSet, err error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	svc := route53.New(sess, aws.NewConfig().WithRegion("us-west-2"))
	zones, err := svc.ListHostedZones(&route53.ListHostedZonesInput{})
	if err != nil {
		return nil, err
	}
	var lock sync.Mutex

	wg := sync.WaitGroup{}
	for _, z := range zones.HostedZones {
		wg.Add(1)
		go func(z *route53.HostedZone) error {
			max := "500"
			recordSets, err := svc.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{HostedZoneId: z.Id, MaxItems: &max})
			if err != nil {
				return err
			}
			var r []RecordSet
			for _, recordSet := range recordSets.ResourceRecordSets {
				if strings.Contains(*recordSet.Name, filter) {
					r = append(r, RecordSet{Zone: z, RecordSet: recordSet})
				}
			}
			lock.Lock()
			records = append(records, r...)
			lock.Unlock()
			wg.Done()
			return nil
		}(z)
	}
	wg.Wait()
	return records, nil
}

func ListAllRecords(filter string) error {
	records, err := SearchRecords(filter)
	if err != nil {
		return err
	}
	details := `{{ "Zone:" | faint }} {{ .Zone.Name }}
{{ "Private:" | faint }} {{ .Zone.Config.PrivateZone }} 
{{ "Name:" | faint }} {{ .RecordSet.Name }} 
{{ "Type:" | faint }} {{ .RecordSet.Type }}
{{ "TTL:" | faint }} {{ if .RecordSet.TTL }}{{ .RecordSet.TTL }}s{{ end }}
{{ "Alias:" | faint }} {{ if .RecordSet.AliasTarget }}{{ .RecordSet.AliasTarget }}{{ end }}
{{ "Value(s):" | faint }}
{{ range $record := .RecordSet.ResourceRecords }}{{ $record.Value }}
{{ end }}`
	templates := &promptui.SelectTemplates{
		Label: "{{ . }}:",
		Active: "▶ {{ .RecordSet.Name }}	{{ .RecordSet.Type }}",
		Inactive: "  {{ .RecordSet.Name }}	{{ .RecordSet.Type }}",
		Selected: "▶ {{ .RecordSet.Name }}	{{ .RecordSet.Type }}",
		Details: details,
	}

	searcher := func(input string, index int) bool {
		i := records[index]
		name := strings.Replace(strings.ToLower(*i.RecordSet.Name+*i.RecordSet.Type), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	_, y, err := utils.GetTerminalSize()
	if err != nil {
		return err
	}
	prompt := promptui.Select{
		Size:      int(y) - 30,
		Label:     "Pick a record",
		Items:     records,
		Templates: templates,
		Searcher:  searcher,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return err
	}
	t, err := template.New("records").Funcs(promptui.FuncMap).Parse(details)
	if err != nil {
		return err
	}
	return t.Execute(os.Stdout, records[i])
}
