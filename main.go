package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/thejerf/suture"
)

var (
	listenAddr   = ":8080"
	githubSecret = ""
)

func main() {
	flag.StringVar(&listenAddr, "listen", listenAddr, "Listen address")
	flag.StringVar(&githubSecret, "secret", githubSecret, "Github secret token")
	flag.Parse()

	h := newWebhook("st-review")
	http.Handle("/", h)

	main := suture.NewSimple("main")
	main.Add(h)
	main.ServeBackground()

	log.Fatal(http.ListenAndServe(listenAddr, nil))
}

func squash(pr int, user, msg string) (string, error) {

	sourceBranch := fmt.Sprintf("pr/%d", pr)
	s := newScript()
	s.run("git", "fetch", "-f", "origin", fmt.Sprintf("refs/pull/%d/head:pr/%d", pr, pr))
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
	if msg == "" {
		// Overridden commit message from parameters
		body = msg
	} else {
		// Commit message from first commit
		body = t.run("git", "log", "-n1", "--pretty=format:%B", sourceBranch)
	}

	body = fmt.Sprintf("%s\n\nMerged by: %s\n", strings.TrimSpace(body), user)

	s.run("git", "merge", "--squash", "--no-commit", sourceBranch)
	s.runPipe(bytes.NewBufferString(msg), "git", "commit", "-F", "-")
	s.run("git", "push", "origin", "master")

	return s.output.String(), s.Error()
}
