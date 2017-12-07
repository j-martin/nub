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

func ListBranches(cfg Configuration) error {
	type branch struct {
		repository, branch, name, email string
		age                             int
	}
	authors := map[string][]branch{}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.GitHub.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	orgOptions := github.RepositoryListByOrgOptions{ListOptions: github.ListOptions{PerPage: 250}}
	repos, _, err := client.Repositories.ListByOrg(ctx, cfg.GitHub.Organization, &orgOptions)
	if err != nil {
		return err
	}
	log.Print(len(repos))
	for _, r := range repos {
		log.Printf("\n-------- %v --------", *r.Name)
		branches, _, err := client.Repositories.ListBranches(ctx, cfg.GitHub.Organization, *r.Name, &github.ListOptions{PerPage: 250})
		if err != nil {
			return err
		}
		for _, b := range branches {
			if *b.Name == "master" {
				continue
			}
			b, _, err := client.Repositories.GetBranch(ctx, cfg.GitHub.Organization, *r.Name, url.PathEscape(*b.Name))
			if err != nil {
				return err
			}
			author := b.Commit.Commit.GetAuthor()
			age := time.Since(*author.Date).Hours() / 24
			if age > 60 {
				br := branch{repository: *r.Name, branch: *b.Name, name: *author.Name, email: *author.Email, age: int(age)}
				authors[br.name] = append(authors[br.name], br)
				log.Printf("%v", br)
			}
		}
	}
	for author, branches := range authors {
		fmt.Println("\n" + author)
		for _, b := range branches {
			fmt.Printf("https://github.com/%v/%v/branches/yours %v\n", cfg.GitHub.Organization, b.repository, b.branch)
		}
	}
	return nil
}
