package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The handler receives commands from the webhook
type handler struct {
	token       string
	username    string
	allowed     []string
	teamAllowed []string
	stop        chan struct{}
	permissions
}

func newHandler(allowed []string, username, token string) *handler {
	return &handler{
		username: username,
		token:    token,
		allowed:  allowed,
		stop:     make(chan struct{}),
		permissions: permissions{
			username:      username,
			token:         token,
			alwaysAllowed: allowed,
			teamMembers:   make(map[string][]string),
		},
	}
}

func (h *handler) handlePullReq(p pr) {
	info, err := os.Stat(filepath.Join(p.Repository.FullName, ".git"))
	if err != nil || !info.IsDir() {
		log.Println("No repo for", p.Repository.FullName)
		return
	}

	cur, err := os.Getwd()
	if err != nil {
		log.Println("No working dir?")
		return
	}

	os.Chdir(p.Repository.FullName)

	switch p.Action {
	case "synchronize", "opened", "reopened":
		updatePR(p.Number)
		p.setStatus(stateSuccess, "st-review", "At your service.", h.username, h.token)
	case "closed":
		closePR(p.Number)
		p.setStatus(stateSuccess, "st-review", "Closed.", h.username, h.token)
	}

	os.Chdir(cur)
}

func (h *handler) handleMerge(c comment) {
	if !h.isAllowed(c.Repository.FullName, c.Sender.Login) {
		msg := fmt.Sprintf("I'm sorry, @%s. I'm afraid I can't do that.", c.Sender.Login)
		c.post(msg, h.username, h.token)
		log.Println("Rejecting request by unknown user", c.Sender.Login)
		return
	}

	log.Printf("Attemping merge of PR %d on %s for %s", c.Issue.Number, c.Repository.FullName, c.Sender.Login)

	info, err := os.Stat(filepath.Join(c.Repository.FullName, ".git"))
	if err != nil || !info.IsDir() {
		log.Println("No repo for", c.Repository.FullName)
		return
	}

	cur, err := os.Getwd()
	if err != nil {
		log.Println("No working dir?")
		return
	}

	overrideDescr := ""
	body := c.parseBody()

	if body.subject != "" {
		overrideDescr = strings.TrimSpace(body.subject + "\n\n" + reflow(body.description, 76))
	}

	user, err := c.user(h.username, h.token)
	if err != nil || user.Email == "" {
		c.post("Merge failed; could net retrieve user information for @"+c.Sender.Login, h.username, h.token)
		log.Printf("Failed merge of PR %d on %s for %s: no user info (%v)", c.Issue.Number, c.Repository.FullName, c.Sender.Login, err)
		return
	}

	pr, err := c.getPR()
	if err != nil {
		log.Println("No pull request:", err)
		return
	}

	pr.setStatus(stateSuccess, "st-review", "Merging...", h.username, h.token)

	os.Chdir(c.Repository.FullName)
	sha1, err := squash(pr, user, overrideDescr)
	os.Chdir(cur)

	if err != nil {
		c.post(err.Error(), h.username, h.token)
		log.Printf("Failed merge of PR %d on %s for %s:\n%s", c.Issue.Number, c.Repository.FullName, c.Sender.Login, err.Error())
		pr.setStatus(stateFailure, "st-review", "Merge failed.", h.username, h.token)

		return
	}

	resMd := fmt.Sprintf("OK, merged as %s. Thanks, @%s!", sha1, c.Issue.User.Login)
	c.post(resMd, h.username, h.token)
	pr.setStatus(stateSuccess, "st-review", "Merged.", h.username, h.token)
	c.close(h.username, h.token)
	log.Printf("Completed merge of PR %d on %s for %s", c.Issue.Number, c.Repository.FullName, c.Sender.Login)
}

var allowedCommitSubjectRe = regexp.MustCompile(`^[a-zA-Z0-9_./-]+:\s`)

func squash(pr pr, user user, msg string) (string, error) {
	sourceBranch := fmt.Sprintf("pr-%d", pr.Number)
	s := newScript()
	s.run("git", "fetch", "-f", "origin", fmt.Sprintf("refs/pull/%d/head:pr-%d", pr.Number, pr.Number))
	s.run("git", "fetch", "-f", "origin", "master:orig/master")

	s.run("git", "reset", "--hard")
	s.run("git", "checkout", "master")
	s.run("git", "reset", "--hard", "orig/master")
	s.run("git", "clean", "-fxd")

	// Find first commit and extract info from it
	t := newScript()
	mergeBase := t.run("git", "merge-base", sourceBranch, "master")
	revs := strings.Fields(t.run("git", "rev-list", mergeBase+".."+sourceBranch))
	if len(revs) == 0 {
		return "", fmt.Errorf("Nothing to merge, as far as I can tell.")
	}
	firstCommit := revs[len(revs)-1]
	authorName := t.run("git", "log", "-n1", "--pretty=format:%an", firstCommit)
	authorEmail := t.run("git", "log", "-n1", "--pretty=format:%ae", firstCommit)
	os.Setenv("GIT_COMMITTER_NAME", user.Name)
	os.Setenv("GIT_COMMITTER_EMAIL", user.Email)
	os.Setenv("GIT_AUTHOR_NAME", authorName)
	os.Setenv("GIT_AUTHOR_EMAIL", authorEmail)

	var body string
	if msg != "" {
		// Overridden commit message from parameters
		body = msg
	} else {
		// Commit message from first commit
		body = t.run("git", "log", "-n1", "--pretty=format:%B", firstCommit)
	}

	body = fmt.Sprintf("%s\n\nGitHub-Pull-Request: %s\n", strings.TrimSpace(body), pr.HTMLURL)

	s.run("git", "merge", "--squash", "--no-commit", sourceBranch)
	s.runPipe(bytes.NewBufferString(body), "git", "commit", "-F", "-")
	sha1 := s.run("git", "rev-parse", "HEAD")
	s.run("git", "push", "origin", "master")

	if s.Error() != nil {
		// Overwrite the error with whatever actual output we had, as a markdown verbatim.
		return "", fmt.Errorf("Merge failed:\n\n```\n%s\n```\n", s.output.String())
	}
	return sha1, nil
}

func updatePR(pr int) {
	s := newScript()
	s.run("git", "fetch", "-f", "origin", fmt.Sprintf("refs/pull/%d/head:pr-%d", pr, pr))
	s.run("git", "push", "-f", "origin", fmt.Sprintf("pr-%d", pr))
}

func closePR(pr int) {
	s := newScript()
	s.run("git", "push", "origin", fmt.Sprintf(":pr-%d", pr))
}
