package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type pr struct {
	Action      string
	Number      int
	PullRequest struct {
		Head struct {
			SHA string
		}
	} `json:"pull_request"`
	Repository struct {
		FullName    string `json:"full_name"`
		StatusesURL string `json:"statuses_url"` // set in events, contains {sha} placeholder
	}
	StatusesURL string   `json:"statuses_url"` // set when getting manually
	HTMLURL     string   `json:"html_url"`     // set when getting manually
	IssueURL    string   `json:"issue_url"`    // set when getting manually
	Base        struct { // set when getting manually
		Ref  string
		Repo struct {
			URL string
		}
	}
}

type prState string

const (
	statePending prState = "pending"
	stateSuccess prState = "success"
	stateError   prState = "error"
	stateFailure prState = "failure"
)

type status struct {
	State   prState
	Context string
	Creator struct {
		Login string
	}
}

func (p *pr) setStatus(state prState, context, description, username, token string) {
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(map[string]string{
		"state":       string(state),
		"description": description,
		"context":     context,
	})

	url := p.StatusesURL
	if url == "" {
		url = p.Repository.StatusesURL
	}
	url = strings.Replace(url, "{sha}", p.PullRequest.Head.SHA, 1)

	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		log.Println("Request:", err)
		return
	}
	req.SetBasicAuth(username, token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Post:", err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode > 299 {
		log.Println("Post:", resp.Status)
		return
	}
}

func (p *pr) setLabel(label, username, token string) {
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode([]string{label})

	req, err := http.NewRequest("POST", p.IssueURL+"/labels", buf)
	if err != nil {
		log.Println("Request:", err)
		return
	}
	req.SetBasicAuth(username, token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Post:", err)
		return
	}
	resp.Body.Close()

	if resp.StatusCode > 299 {
		log.Println("Post:", resp.Status)
		return
	}
}

type branch struct {
	Protection struct {
		Enabled bool
		Checks  struct {
			Contexts []string
		} `json:"required_status_checks"`
	}
}

func (p *pr) getRequiredStatuses(username, token string) []string {
	url := p.Base.Repo.URL + "/branches/" + p.Base.Ref
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Request:", err)
		return nil
	}
	req.SetBasicAuth(username, token)
	req.Header.Set("Accept", "application/vnd.github.loki-preview")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Get:", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		log.Println("Get:", resp.Status)
		return nil
	}

	var tmp branch
	if err := json.NewDecoder(resp.Body).Decode(&tmp); err != nil {
		log.Println("JSON:", err)
		return nil
	}

	if !tmp.Protection.Enabled {
		return nil
	}

	log.Println("Required checks:", tmp.Protection.Checks.Contexts)
	return tmp.Protection.Checks.Contexts
}

func (p *pr) getStatuses(username, token string) []status {
	req, err := http.NewRequest("GET", p.StatusesURL, nil)
	if err != nil {
		log.Println("Request:", err)
		return nil
	}
	req.SetBasicAuth(username, token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Get:", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		log.Println("Post:", resp.Status)
		return nil
	}

	var tmp []status
	if err := json.NewDecoder(resp.Body).Decode(&tmp); err != nil {
		log.Println("JSON:", err)
		return nil
	}

	// We only want the latest status for each context, which comes earlier
	// in the list from GitHub
	var res []status
	seen := make(map[string]bool)
	for _, s := range tmp {
		if seen[s.Context] {
			continue
		}
		res = append(res, s)
		seen[s.Context] = true
	}

	return res
}

func overallStatus(ss []status, skip []string, req []string) prState {
	total := stateSuccess

	skipContext := make(map[string]bool)
	var skipREs []*regexp.Regexp
	for _, s := range skip {
		if re, err := regexp.Compile(strings.ToLower(s)); err == nil {
			skipREs = append(skipREs, re)
		}
		skipContext[s] = true
	}

	reqContext := make(map[string]bool)
	reqPending := make(map[string]struct{})
	for _, s := range req {
		reqContext[s] = true
		reqPending[s] = struct{}{}
	}

outer:
	for _, s := range ss {
		delete(reqPending, s.Context)

		if skipContext[s.Context] {
			continue
		}
		for _, re := range skipREs {
			if re.MatchString(strings.ToLower(s.Context)) {
				continue outer
			}
		}

		if len(req) > 0 && !reqContext[s.Context] {
			continue
		}

		switch s.State {
		case stateError, stateFailure:
			return s.State
		case statePending:
			if total == stateSuccess {
				total = statePending
			}
		}
	}

	if len(reqPending) > 0 && total == stateSuccess {
		return statePending
	}

	return total
}
