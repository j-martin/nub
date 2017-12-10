package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"log"
	"net/url"
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
