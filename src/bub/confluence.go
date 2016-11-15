package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/bndr/gopencils"
	"github.com/russross/blackfriday"
	"strings"
)

type PageInfo struct {
	Title     string `json:"title"`

	Version   struct {
				  Number int64 `json:"number"`
			  } `json:"version"`

	Ancestors []struct {
		Id string `json:"id"`
	} `json:"ancestors"`
}

func createPage() []byte {
	markdownData := []byte("```\nThis page and its children are automatically generated. Any changes will be lost.\n```\n")
	for _, document := range []string{"README", "CHANGELOG"} {
		filename := document + ".md"
		content, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Printf("Update Page: %v not found. Continuing.", filename)
		} else {
			markdownData = append(markdownData, "# " + strings.Title(strings.ToLower(document)) + "\n"...)
			markdownData = append(markdownData, content...)
			markdownData = append(markdownData, "\n"...)
		}
	}

	return blackfriday.MarkdownCommon(markdownData)
}

func UpdateDocumentation(m Manifest) {

	if m.Page == "" {
		log.Print("Update Page: No confluence page defined in manifest. Moving on.")
		return
	}

	htmlData := createPage()

	username := os.Getenv("CONFLUENCE_USER")
	password := os.Getenv("CONFLUENCE_PASSWORD")

	api := gopencils.Api(
		"https://example.atlassian.net/wiki/rest/api",
		&gopencils.BasicAuth{Username: username, Password: password},
	)

	pageInfo, err := getPageInfo(api, m.Page)
	if err != nil {
		log.Fatal(err)
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
		{"id": pageInfo.Ancestors[len(pageInfo.Ancestors) - 1].Id},
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
		"content/" + pageID, &map[string]interface{}{},
	).Put(payload)
	if err != nil {
		return err
	}

	if request.Raw.StatusCode != 200 {
		output, _ := ioutil.ReadAll(request.Raw.Body)
		defer request.Raw.Body.Close()

		return fmt.Errorf(
			"Confluence REST API returns unexpected HTTP status: %s, " +
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
		"content/" + pageID, &PageInfo{},
	).Get(map[string]string{"expand": "ancestors,version"})

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
