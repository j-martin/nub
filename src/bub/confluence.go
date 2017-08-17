package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"bufio"
	"bytes"
	"github.com/bndr/gopencils"
	"github.com/russross/blackfriday"
	"gopkg.in/yaml.v2"
	"strings"
	"text/template"
)

type PageInfo struct {
	Title string   `json:"title"`
	Body  PageBody `json:"body"`

	Version struct {
		Number int64 `json:"number"`
	} `json:"version"`

	Ancestors []struct {
		Id string `json:"id"`
	} `json:"ancestors"`
}

type PageBody struct {
	Storage PageBodyValue `json:"storage"`
}

type PageBodyValue struct {
	Value string `json:"value"`
}

type PageParams struct {
	Manifest   string
	Repository string
	Name       string
}

func shortManifest(m Manifest) ([]byte, error) {
	m.Readme = "See below."
	m.ChangeLog = "See below."
	m.Branch = ""
	m.LastUpdate = 0
	m.Version = ""
	return yaml.Marshal(m)
}

func createPage(m Manifest) []byte {
	yml, _ := shortManifest(m)
	t, err := template.New("readme").Parse(
		"[Repository](https://github.com/BenchLabs/{{.Repository}}) | Diffs " +
			"[Production / Master](https://github.com/BenchLabs/{{.Repository}}/compare/production...master) / " +
			"[Staging / Production](https://github.com/BenchLabs/{{.Repository}}/compare/production...staging) / " +
			"[Previous / Current Production](https://github.com/BenchLabs/{{.Repository}}/compare/production-rollback...production) | " +
			"[Jenkins](https://jenkins.example.com/job/BenchLabs/job/{{.Repository}}) | " +
			"[Splunk](https://splunk.example.com/en-US/app/search/search/?dispatch.sample_ratio=1&earliest=rt-1h&latest=rtnow&q=search%20sourcetype%3Dpro-{{.Name}}-hec&display.page.search.mode=smart)\n\n" +
			"This page is automatically generated. Any changes will be lost.\n" +
			"```\n{{.Manifest}}\n```\n")

	if err != nil {
		log.Fatal(err)
	}
	var templated bytes.Buffer
	writer := bufio.NewWriter(&templated)
	err = t.Execute(writer, PageParams{string(yml), m.Repository, m.Name})
	writer.Flush()

	if err != nil {
		log.Fatal(err)
	}

	markdown := append(templated.Bytes(), m.Readme+"\n"...)
	markdown = append(markdown, m.ChangeLog+"\n"...)

	htmlFlags := blackfriday.HTML_USE_XHTML
	renderer := blackfriday.HtmlRenderer(htmlFlags, "", "")

	extensions := 0 |
		blackfriday.EXTENSION_FENCED_CODE |
		blackfriday.EXTENSION_TABLES |
		blackfriday.EXTENSION_AUTOLINK |
		blackfriday.EXTENSION_BACKSLASH_LINE_BREAK |
		blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
		blackfriday.EXTENSION_DEFINITION_LISTS

	opts := blackfriday.Options{Extensions: extensions}
	return blackfriday.MarkdownOptions(markdown, renderer, opts)
}

func UpdateDocumentation(cfg Configuration, m Manifest) {

	if m.Page == "" {
		log.Print("Page: No confluence page defined in manifest. Moving on.")
		return
	}

	htmlData := createPage(m)
	newContent := string(htmlData[:])

	username := os.Getenv("CONFLUENCE_USER")
	if username == "" {
		username = cfg.Confluence.Username
	}

	password := os.Getenv("CONFLUENCE_PASSWORD")
	if password == "" {
		password = cfg.Confluence.Password
	}

	api := gopencils.Api(
		cfg.Confluence.Server+"/rest/api",
		&gopencils.BasicAuth{Username: username, Password: password},
	)

	pageInfo, err := getPageInfo(api, m.Page)
	pageInfo.Title = strings.Title(m.Name) + " - Readme"
	if err != nil {
		log.Fatal(err)
	}

	currentBody := pageInfo.Body.Storage.Value
	if strings.Contains(newContent, currentBody) {
		log.Print("No update needed. Skipping.")
		return
	}

	err = updatePage(api, m.Page, pageInfo, htmlData)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Page successfully updated.")
}

func updatePage(
	api *gopencils.Resource, pageID string,
	pageInfo PageInfo, newContent []byte,
) error {
	nextPageVersion := pageInfo.Version.Number + 1

	if len(pageInfo.Ancestors) == 0 {
		return fmt.Errorf(
			"Page '%s' info does not contain any information about parents",
			pageID,
		)
	}

	// picking only the last one, which is required by confluence
	oldAncestors := []map[string]interface{}{
		{"id": pageInfo.Ancestors[len(pageInfo.Ancestors)-1].Id},
	}

	payload := map[string]interface{}{
		"id":    pageID,
		"type":  "page",
		"title": pageInfo.Title,
		"version": map[string]interface{}{
			"number":    nextPageVersion,
			"minorEdit": false,
		},
		"ancestors": oldAncestors,
		"body": map[string]interface{}{
			"storage": map[string]interface{}{
				"value":          string(newContent),
				"representation": "storage",
			},
		},
	}

	request, err := api.Res(
		"content/"+pageID, &map[string]interface{}{},
	).Put(payload)
	if err != nil {
		return err
	}

	if request.Raw.StatusCode != 200 {
		output, _ := ioutil.ReadAll(request.Raw.Body)
		defer request.Raw.Body.Close()

		return fmt.Errorf(
			"Confluence REST API returns unexpected HTTP status: %s, "+
				"output: %s",
			request.Raw.Status, output,
		)
	}

	return nil
}

func getPageInfo(
	api *gopencils.Resource, pageID string,
) (PageInfo, error) {
	request, err := api.Res(
		"content/"+pageID, &PageInfo{},
	).Get(map[string]string{"expand": "body.storage,ancestors,version"})

	if err != nil {
		return PageInfo{}, err
	}

	if request.Raw.StatusCode == 401 {
		return PageInfo{}, fmt.Errorf("authentification failed")
	}

	if request.Raw.StatusCode == 404 {
		return PageInfo{}, fmt.Errorf(
			"page with id '%s' not found, Confluence REST API returns 404",
			pageID,
		)
	}

	if request.Raw.StatusCode != 200 {
		return PageInfo{}, fmt.Errorf(
			"Confluence REST API returns unexpected HTTP status: %s",
			request.Raw.Status,
		)
	}

	response := request.Response.(*PageInfo)

	return *response, nil
}
