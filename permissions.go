package main

import (
	"context"
	"log"
	"sort"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type permissions struct {
	token         string
	alwaysAllowed []string
	teamMembers   map[string][]string // repo -> list of members
}

func (p *permissions) isAllowed(repo, login string) bool {
	// Check the list of always allowed users
	for _, user := range p.alwaysAllowed {
		if login == user {
			return true
		}
	}

	// Check the cached list of team members for the given repo
	for _, user := range p.teamMembers[repo] {
		if login == user {
			return true
		}
	}

	// Refresh the team members list as it may be out of date
	log.Println("Refreshing the list of collaborators on", repo, "...")
	users, err := p.collaborators(repo)
	if err != nil {
		return false
	}
	log.Println(" ... got", users)
	p.teamMembers[repo] = users
	for _, user := range p.teamMembers[repo] {
		if login == user {
			return true
		}
	}

	log.Println("Permission denied for", login)

	// Nope, no match
	return false
}

func (p *permissions) collaborators(repo string) ([]string, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: p.token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)

	opt := new(github.ListCollaboratorsOptions)
	opt.PerPage = 50
	var allCollabs []*github.User
	ps := strings.Split(repo, "/")
	owner, repo := ps[0], ps[1]
	for {
		users, resp, err := client.Repositories.ListCollaborators(context.TODO(), owner, repo, opt)
		if err != nil {
			return nil, err
		}
		allCollabs = append(allCollabs, users...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	userMap := make(map[string]bool)
	for _, user := range allCollabs {
		userMap[*user.Login] = true
	}

	var users []string
	for user := range userMap {
		users = append(users, user)
	}
	sort.Strings(users)
	return users, nil
}
