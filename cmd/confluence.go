package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"bufio"
	"bytes"
	"github.com/bndr/gopencils"
	"github.com/pkg/errors"
	"github.com/russross/blackfriday"
	"gopkg.in/yaml.v2"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

type Confluence struct {
	cfg    *Configuration
	client *gopencils.Resource
}

func MustInitConfluence(cfg *Configuration) *Confluence {
	api := gopencils.Api(
		cfg.Confluence.Server+"/rest/api",
		&gopencils.BasicAuth{Username: cfg.Confluence.Username, Password: cfg.Confluence.Password},
	)
	return &Confluence{client: api, cfg: cfg}
}

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

func (c *Confluence) marshallManifest(m Manifest) (string, error) {
	m.Readme = "See below."
	m.ChangeLog = "See below."
	m.Branch = ""
	m.LastUpdate = 0
	m.Version = ""
	i, err := yaml.Marshal(m)
	return string(i), err
}

func (c *Confluence) findMarkdownFiles(ignoreDirs []string, ignoreCommonFiles bool) (fileList []string, err error) {
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

func (c *Confluence) joinMarkdownFiles(m Manifest) (content []byte, err error) {
	files, err := c.findMarkdownFiles(m.Documentation.IgnoredDirs, true)
	if err != nil {
		return nil, err
	}
	for _, filePath := range files {
		fileContent, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		url := c.generateGitHubLink(filePath, m)
		content = append(content, []byte("\n\n\n---\n#### From "+url+"\n")...)
		content = append(content, fileContent...)
	}
	return content, err
}

func (c *Confluence) generateGitHubLink(filePath string, m Manifest) string {
	return "[" + filePath + "](https://github.com/" + path.Join(c.cfg.GitHub.Organization, m.Repository, "blob/master", filePath) + ")"
}

func (c *Confluence) createPage(m Manifest) ([]byte, error) {
	yaml, err := c.marshallManifest(m)
	if err != nil {
		return nil, err
	}

	t, err := template.New("readme").Parse(`
<ac:structured-macro ac:name="info" ac:schema-version="1" ac:macro-id="9289e233-4abf-4957-8884-bef7be9ead8e"><ac:rich-text-body>
<p>This page is automatically generated. Any changes will be lost.
	Edit the actual <a href="https://github.com/{{ .Config.GitHub.Organization }}/{{ .Manifest.Repository }}">README</a> instead.</p>
</ac:rich-text-body></ac:structured-macro>

<p>
	<a href="https://github.com/{{ .Config.GitHub.Organization }}/{{ .Manifest.Repository }}">Repository</a> |
	<strong>Diffs</strong>
		<a href="https://github.com/{{ .Config.GitHub.Organization }}/{{ .Manifest.Repository }}/compare/production...master" title="Pending changes from master to Production">Production / Master</a> /
		<a href="https://github.com/{{ .Config.GitHub.Organization }}/{{ .Manifest.Repository }}/compare/production...staging" title="Pending changes from Staging to Production">Staging / Production</a> /
		<a href="https://github.com/{{ .Config.GitHub.Organization }}/{{ .Manifest.Repository }}/compare/production-rollback...production" title="Changes in the previous deployment.">Previous / Current Production</a> |
	<a href="{{ .Config.Jenkins.Server }}/job/{{ .Config.GitHub.Organization }}/job/{{ .Manifest.Repository }}">Jenkins</a> |
	<a href="{{ .Config.Splunk.Server }}/en-US/app/search/search/?dispatch.sample_ratio=1&amp;earliest=rt-1h&amp;latest=rtnow&amp;q=search%20sourcetype%3D{{ .Manifest.Deploy.Environment }}-{{ .Manifest.Name }}*&amp;display.page.search.mode=smart">Splunk</a>
</p>

<p>
	<ac:structured-macro ac:name="expand" ac:schema-version="1" ac:macro-id="856ee728-b2f6-4c39-b63d-e1e4a2b9a6ed">
		<ac:parameter ac:name="title">See manifest...</ac:parameter><ac:rich-text-body>
		<ac:structured-macro ac:name="code" ac:schema-version="1" ac:macro-id="9d13770a-90d2-4283-93fc-3faf24eef746"><ac:plain-text-body>
			<![CDATA[{{ .YAML }}]]>
		</ac:plain-text-body></ac:structured-macro>
	</ac:rich-text-body></ac:structured-macro>
</p>
`)

	if err != nil {
		return nil, err
	}
	var header bytes.Buffer
	writer := bufio.NewWriter(&header)
	err = t.Execute(writer, struct {
		Config   Configuration
		Manifest Manifest
		YAML     string
	}{
		Manifest: m,
		Config:   *c.cfg,
		YAML:     yaml,
	})
	writer.Flush()

	if err != nil {
		log.Fatal(err)
	}

	otherMarkdown, err := c.joinMarkdownFiles(m)
	if err != nil {
		return nil, err
	}
	markdown := append([]byte(m.Readme), otherMarkdown...)

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
	renderedMarkdown := blackfriday.MarkdownOptions(markdown, renderer, opts)
	return append(header.Bytes(), renderedMarkdown...), nil
}

func (c *Confluence) updateDocumentation(m Manifest) error {
	loadCredentials("Confluence", &c.cfg.Confluence)

	if m.Documentation.PageId == "" {
		log.Print("documenation.pageId: No confluence page defined in manifest. Moving on.")
		return nil
	}

	htmlData, err := c.createPage(m)
	if err != nil {
		return errors.Errorf("Failed to generate page. %v", err)
	}
	newContent := string(htmlData[:])

	pageInfo, err := c.getPageInfo(m.Documentation.PageId)
	pageInfo.Title = strings.Title(m.Name) + " - Readme"
	if err != nil {
		return err
	}

	currentBody := pageInfo.Body.Storage.Value
	if sanitizeBody(newContent) == sanitizeBody(currentBody) {
		log.Print("No update needed. Skipping.")
		return nil
	}

	err = c.updatePage(m.Documentation.PageId, pageInfo, htmlData)
	if err != nil {
		return err
	}

	log.Println("Page successfully updated.")
	return nil
}

func sanitizeBody(body string) string {
	r := strings.NewReplacer(" ", "", "\n", "", "\t", "")
	return r.Replace(body)
}

func (c *Confluence) updatePage(pageID string, pageInfo PageInfo, newContent []byte) error {
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

	request, err := c.client.Res(
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

func (c *Confluence) getPageInfo(pageID string) (PageInfo, error) {
	request, err := c.client.Res(
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
