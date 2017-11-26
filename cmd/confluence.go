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
	"path"
	"path/filepath"
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

func shortManifest(m Manifest) ([]byte, error) {
	m.Readme = "See below."
	m.ChangeLog = "See below."
	m.Branch = ""
	m.LastUpdate = 0
	m.Version = ""
	return yaml.Marshal(m)
}

func findMarkdownFiles(ignoreDirs []string, ignoreCommonFiles bool) (fileList []string, err error) {
	err = filepath.Walk(".", func(filePath string, f os.FileInfo, err error) error {
		ignoredRootDir := append(ignoreDirs,
			".github",
			"bower_components/",
			"node_modules/",
			"pkg/",
			".repositories/",
			"vendor/",
			"PULL_REQUEST_TEMPLATE.md",
		)

		for _, dir := range ignoredRootDir {
			if strings.HasPrefix(filePath, dir) {
				return nil
			}
		}
		commonIgnoredDirs := []string{
			"bower_components/",
			"node_modules/",
		}
		for _, dir := range commonIgnoredDirs {
			if strings.Contains(filePath, dir) {
				return nil
			}
		}

		if ignoreCommonFiles {
			ignoredFiles := []string{
				"README.md",
			}
			for _, ignoredFile := range ignoredFiles {
				if filePath == ignoredFile {
					return nil
				}
			}

		}
		if path.Ext(filePath) == ".md" {
			fileList = append(fileList, filePath)
		}
		return nil
	})
	return fileList, err
}

func joinMarkdownFiles(m Manifest) (content []byte, err error) {
	files, err := findMarkdownFiles(m.Documentation.IgnoredDirs, true)
	if err != nil {
		return nil, err
	}
	for _, filePath := range files {
		fileContent, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		url := generateGitHubLink(filePath, m)
		content = append(content, []byte("\n\n\n---\n#### From "+url+"\n")...)
		content = append(content, fileContent...)
	}
	return content, err
}
func generateGitHubLink(filePath string, m Manifest) string {
	return "[" + filePath + "](https://github.com/BenchLabs/" + path.Join(m.Repository, "blob/master", filePath) + ")"
}

func createPage(m Manifest) ([]byte, error) {
	t, err := template.New("readme").Parse(`
[Repository](https://github.com/BenchLabs/{{.Repository}}) | **Diffs**
[Production / Master](https://github.com/BenchLabs/{{.Repository}}/compare/production...master) /
[Staging / Production](https://github.com/BenchLabs/{{.Repository}}/compare/production...staging) /
[Previous / Current Production](https://github.com/BenchLabs/{{.Repository}}/compare/production-rollback...production) |
[Jenkins](https://jenkins.example.com/job/BenchLabs/job/{{.Repository}}) |
[Splunk](https://splunk.example.com/en-US/app/search/search/?dispatch.sample_ratio=1&earliest=rt-1h&latest=rtnow&q=search%20sourcetype%3D{{.Deploy.Environment}}-{{.Name}}*&display.page.search.mode=smart)


`)
	renderedBody := []byte(`
<ac:structured-macro ac:name="info" ac:schema-version="1" ac:macro-id="9289e233-4abf-4957-8884-bef7be9ead8e"><ac:rich-text-body>
<p>This page is automatically generated. Any changes will be lost.</p>
</ac:rich-text-body></ac:structured-macro>
`)
	manifestHeader := []byte(`
<ac:structured-macro ac:name="expand" ac:schema-version="1" ac:macro-id="856ee728-b2f6-4c39-b63d-e1e4a2b9a6ed"><ac:parameter ac:name="title">See manifest...</ac:parameter><ac:rich-text-body>
<ac:structured-macro ac:name="code" ac:schema-version="1" ac:macro-id="9d13770a-90d2-4283-93fc-3faf24eef746"><ac:plain-text-body><![CDATA[
`)
	manifestFooter := []byte(`
]]></ac:plain-text-body></ac:structured-macro>
</ac:rich-text-body></ac:structured-macro> `)

	manifestBytes, _ := shortManifest(m)
	renderedBody = append(renderedBody, manifestHeader...)
	renderedBody = append(renderedBody, manifestBytes...)
	renderedBody = append(renderedBody, manifestFooter...)
	if err != nil {
		return nil, err
	}
	var templated bytes.Buffer
	writer := bufio.NewWriter(&templated)
	err = t.Execute(writer, m)
	writer.Flush()

	if err != nil {
		log.Fatal(err)
	}

	markdown := append(templated.Bytes(), m.Readme+"\n"...)
	otherDocs, err := joinMarkdownFiles(m)
	if err != nil {
		return nil, err
	}
	markdown = append(markdown, otherDocs...)

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
	renderedBody = append(renderedBody, blackfriday.MarkdownOptions(markdown, renderer, opts)...)
	return renderedBody, nil
}

func updateDocumentation(cfg Configuration, m Manifest) {

	if m.Documentation.PageId == "" {
		log.Print("documenation.pageId: No confluence page defined in manifest. Moving on.")
		return
	}

	htmlData, err := createPage(m)
	if err != nil {
		log.Fatalf("Failed to generate page. %v", err)
	}
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

	pageInfo, err := getPageInfo(api, m.Documentation.PageId)
	pageInfo.Title = strings.Title(m.Name) + " - Readme"
	if err != nil {
		log.Fatal(err)
	}

	currentBody := pageInfo.Body.Storage.Value
	if newContent == currentBody {
		log.Print("No update needed. Skipping.")
		return
	}

	err = updatePage(api, m.Documentation.PageId, pageInfo, htmlData)
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
			"page '%s' info does not contain any information about parents",
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

		log.Printf(string(newContent))
		return fmt.Errorf(
			"confluence REST API returns unexpected HTTP status: %s, "+
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
			"confluence REST API returns unexpected HTTP status: %s",
			request.Raw.Status,
		)
	}

	response := request.Response.(*PageInfo)

	return *response, nil
}
