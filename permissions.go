package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
)

type permissions struct {
	username      string
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
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/collaborators", repo), nil)
	if err != nil {
		log.Println("Request:", err)
		return nil, err
	}
	req.SetBasicAuth(p.username, p.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Get:", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		log.Println("Get:", resp.Status)
		return nil, err
	}

	var res []user
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, err
	}

	userMap := make(map[string]bool)
	for _, user := range res {
		userMap[user.Login] = true
	}

	var users []string
	for user := range userMap {
		users = append(users, user)
	}
	sort.Strings(users)
	return users, nil
}
