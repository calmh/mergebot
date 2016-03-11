package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// The handler receives commands from the webhook
type handler struct {
	token    string
	username string
	allowed  []string
	stop     chan struct{}
}

func newHandler(allowed []string, username, token string) *handler {
	return &handler{
		username: username,
		token:    token,
		allowed:  allowed,
		stop:     make(chan struct{}),
	}
}

func (h *handler) isAllowed(user string) bool {
	for _, allowed := range h.allowed {
		if user == allowed {
			return true
		}
	}
	return false
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
	case "closed":
		closePR(p.Number)
	}

	os.Chdir(cur)
}

func (h *handler) handleMerge(c comment) {
	if !h.isAllowed(c.Sender.Login) {
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

	os.Chdir(c.Repository.FullName)
	res, sha1, err := squash(c.Issue.Number, user, overrideDescr)
	os.Chdir(cur)

	if err != nil {
		resMd := fmt.Sprintf("Merge failed:\n\n```\n%s\n```", res)
		c.post(resMd, h.username, h.token)
		log.Printf("Failed merge of PR %d on %s for %s:\n%s", c.Issue.Number, c.Repository.FullName, c.Sender.Login, res)

		return
	}

	resMd := fmt.Sprintf("OK, merged as %s. Thanks, @%s!", sha1, c.Issue.User.Login)
	c.post(resMd, h.username, h.token)
	c.close(h.username, h.token)
	log.Printf("Completed merge of PR %d on %s for %s", c.Issue.Number, c.Repository.FullName, c.Sender.Login)
}

func squash(pr int, user user, msg string) (string, string, error) {
	sourceBranch := fmt.Sprintf("pr-%d", pr)
	s := newScript()
	s.run("git", "fetch", "-f", "origin", fmt.Sprintf("refs/pull/%d/head:pr-%d", pr, pr))
	s.run("git", "fetch", "-f", "origin", "master:orig/master")

	s.run("git", "reset", "--hard")
	s.run("git", "checkout", "master")
	s.run("git", "reset", "--hard", "orig/master")
	s.run("git", "clean", "-fxd")

	// Find first commit and extract info from it
	t := newScript()
	mergeBase := t.run("git", "merge-base", sourceBranch, "master")
	revs := strings.Fields(t.run("git", "rev-list", mergeBase+".."+sourceBranch))
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

	s.run("git", "merge", "--squash", "--no-commit", sourceBranch)
	s.runPipe(bytes.NewBufferString(body), "git", "commit", "-F", "-")
	sha1 := s.run("git", "rev-parse", "HEAD")
	s.run("git", "push", "origin", "master")

	return s.output.String(), sha1, s.Error()
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
