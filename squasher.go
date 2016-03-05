package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// The squasher receives commands from the inbox and processes them as squashes.
type squasher struct {
	comments <-chan comment
	pullReqs <-chan pr
	token    string
	username string
	allowed  []string
	stop     chan struct{}
}

func newSquasher(comments <-chan comment, pullReqs <-chan pr, allowed []string, username, token string) *squasher {
	return &squasher{
		comments: comments,
		pullReqs: pullReqs,
		username: username,
		token:    token,
		allowed:  allowed,
		stop:     make(chan struct{}),
	}
}

func (h *squasher) Serve() {
	allowed := make(map[string]bool)
	for _, user := range h.allowed {
		allowed[user] = true
	}

	for {
		select {
		case c := <-h.comments:
			if !allowed[c.Sender.Login] {
				msg := fmt.Sprintf("Sorry @%s, I don't know you well enough to allow that.", c.Sender.Login)
				c.post(msg, h.username, h.token)
				log.Println("Rejecting request by unknown user", c.Sender.Login)
				continue
			}

			body := c.parseBody()

			if strings.HasPrefix(body.command, "merge") || strings.HasPrefix(body.command, "squash") {
				log.Printf("Attemping merge of PR %d on %s for %s", c.Issue.Number, c.Repository.FullName, c.Sender.Login)

				info, err := os.Stat(filepath.Join(c.Repository.FullName, ".git"))
				if err != nil || !info.IsDir() {
					log.Println("No repo for", c.Repository.FullName)
					continue
				}

				cur, err := os.Getwd()
				if err != nil {
					log.Println("No working dir?")
					continue
				}

				overrideDescr := ""
				if body.subject != "" {
					overrideDescr = strings.TrimSpace(body.subject + "\n\n" + reflow(body.description, 76))
				}

				os.Chdir(c.Repository.FullName)
				res, sha1, err := squash(c.Issue.Number, c.Sender.Login, overrideDescr)
				os.Chdir(cur)

				if err != nil {
					resMd := fmt.Sprintf("Merge failed:\n\n```\n%s\n```", res)
					c.post(resMd, h.username, h.token)
					log.Printf("Failed merge of PR %d on %s for %s:\n%s", c.Issue.Number, c.Repository.FullName, c.Sender.Login, res)

					continue
				}

				resMd := fmt.Sprintf("OK, merged as %s.", sha1)
				c.post(resMd, h.username, h.token)
				c.close(h.username, h.token)
				log.Printf("Completed merge of PR %d on %s for %s", c.Issue.Number, c.Repository.FullName, c.Sender.Login)
			}

		case p := <-h.pullReqs:
			info, err := os.Stat(filepath.Join(p.Repository.FullName, ".git"))
			if err != nil || !info.IsDir() {
				log.Println("No repo for", p.Repository.FullName)
				continue
			}

			cur, err := os.Getwd()
			if err != nil {
				log.Println("No working dir?")
				continue
			}

			os.Chdir(p.Repository.FullName)

			switch p.Action {
			case "synchronize", "opened", "reopened":
				updatePR(p.Number)
			case "closed":
				closePR(p.Number)
			}

			os.Chdir(cur)

		case <-h.stop:
			return
		}
	}
}

func (h *squasher) Stop() {
	close(h.stop)
}

func squash(pr int, user, msg string) (string, string, error) {
	sourceBranch := fmt.Sprintf("pr-%d", pr)
	s := newScript()
	s.run("git", "fetch", "-f", "origin", fmt.Sprintf("refs/pull/%d/head:pr-%d", pr, pr))
	s.run("git", "fetch", "-f", "origin", "master:orig/master")

	s.run("git", "reset", "--hard")
	s.run("git", "checkout", "master")
	s.run("git", "reset", "--hard", "orig/master")
	s.run("git", "clean", "-fxd")

	t := newScript()
	authorName := t.run("git", "log", "-n1", "--pretty=format:%an", sourceBranch)
	authorEmail := t.run("git", "log", "-n1", "--pretty=format:%ae", sourceBranch)
	os.Setenv("GIT_COMMITTER_NAME", authorName)
	os.Setenv("GIT_COMMITTER_EMAIL", authorEmail)
	os.Setenv("GIT_AUTHOR_NAME", authorName)
	os.Setenv("GIT_AUTHOR_EMAIL", authorEmail)

	var body string
	if msg != "" {
		// Overridden commit message from parameters
		body = msg
	} else {
		// Commit message from first commit
		body = t.run("git", "log", "-n1", "--pretty=format:%B", sourceBranch)
	}

	body = fmt.Sprintf("%s\n\nMerged-by: %s (for PR #%d)\n", strings.TrimSpace(body), user, pr)

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
