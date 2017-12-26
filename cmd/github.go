package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"log"
	"net/url"
	"strings"
	"time"
)

type GitHub struct {
	cfg    *Configuration
	client *github.Client
}

func MustInitGitHub(cfg *Configuration) *GitHub {
	ctx := context.Background()
	loadKeyringItem("GitHub Token", &cfg.GitHub.Token)
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.GitHub.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	return &GitHub{cfg, client}
}

func (gh *GitHub) CreatePR(title, body string) error {
	MustInitGit().Push(gh.cfg)
	MustInitGit().Fetch()
	branch := MustInitGit().GetCurrentBranch()
	base := "master"
	if title == "" {
		subjects := MustInitGit().LogNotInMasterSubjects()
		if len(subjects) == 1 {
			title = subjects[0]
		} else {
			title = strings.Replace(strings.Replace(strings.Replace(branch, "-", "_", 1), "-", " ", -1), "_", "-", -1)
		}
	}

	if body == "" {
		body = MustInitGit().LogNotInMasterBody()
	}

	ctx := context.Background()
	org := gh.cfg.GitHub.Organization
	repo := MustInitGit().GetCurrentRepositoryName()

	request := github.NewPullRequest{Head: &branch, Base: &base, Title: &title, Body: &body}
	pr, _, err := gh.client.PullRequests.Create(ctx, org, repo, &request)

	if err != nil {
		prListOptions := github.PullRequestListOptions{Head: branch, Base: base}
		existingPRs, _, err := gh.client.PullRequests.List(ctx, org, repo, &prListOptions)
		if len(existingPRs) > 0 {
			log.Print("Existing PR found.")
			return OpenURI(*existingPRs[0].HTMLURL)
		}
		return err
	}

	reviewers := gh.cfg.GitHub.Reviewers
	if len(reviewers) > 0 {
		reviewersRequest := github.ReviewersRequest{Reviewers: reviewers}
		pr, _, err = gh.client.PullRequests.RequestReviewers(ctx, org, repo, *pr.Number, reviewersRequest)

		if err != nil {
			return err
		}

	}
	return OpenURI(*pr.HTMLURL)
}

func (gh *GitHub) OpenPage(m *Manifest, p ...string) error {
	base := []string{
		"https://github.com",
		gh.cfg.GitHub.Organization,
		m.Repository,
	}
	return OpenURI(append(base, p...)...)
}

func (gh *GitHub) OpenPR(m *Manifest, pr string) error {
	return gh.OpenPage(m, "pull", pr, "files")
}

func (gh *GitHub) ListBranches(maxAge int) error {
	type branch struct {
		Repository, Branch, Name, Email, PRURL string
		Age                                    int
	}
	ctx := context.Background()
	authors := map[string][]branch{}

	orgOptions := github.RepositoryListByOrgOptions{ListOptions: github.ListOptions{PerPage: 250}}
	org := gh.cfg.GitHub.Organization
	repos, _, err := gh.client.Repositories.ListByOrg(ctx, org, &orgOptions)
	if err != nil {
		return err
	}
	for _, r := range repos {
		log.Print(*r.Name)
		if *r.Fork {
			continue
		}
		branches, _, err := gh.client.Repositories.ListBranches(ctx, org, *r.Name, &github.ListOptions{PerPage: 250})
		if err != nil {
			return err
		}
		prs, _, err := gh.client.PullRequests.List(ctx, org, *r.Name, &github.PullRequestListOptions{State: "open"})
		if err != nil {
			return err
		}
		pullRequests := map[string]*github.PullRequest{}
		for _, pr := range prs {
			pullRequests[*pr.Head.SHA] = pr
		}
		for _, b := range branches {
			if *b.Name == "master" {
				continue
			}
			b, _, err := gh.client.Repositories.GetBranch(ctx, org, *r.Name, url.PathEscape(*b.Name))
			if err != nil {
				return err
			}
			author := b.Commit.Commit.GetAuthor()
			age := int(time.Since(*author.Date).Hours() / 24)
			if age > maxAge {
				sha := *b.Commit.SHA
				pr := pullRequests[sha]
				prURL := ""
				if pr != nil {
					prURL = *pr.HTMLURL
				}
				br := branch{Repository: *r.Name, Branch: *b.Name, Name: *author.Name, Email: *author.Email, Age: int(age), PRURL: prURL}
				authors[br.Name] = append(authors[br.Name], br)
			}
		}
	}
	for author, branches := range authors {
		fmt.Println("\n" + author)
		for _, b := range branches {
			fmt.Printf("https://github.com/%v/%v/branches/yours %v %v\n", org, b.Repository, b.Branch, b.PRURL)
		}
	}
	return nil
}
