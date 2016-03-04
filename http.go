package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type webhook struct {
	username      string
	issueComments chan issueComment
	stop          chan struct{}
}

func newWebhook(username string) *webhook {
	return &webhook{
		username:      username,
		issueComments: make(chan issueComment),
		stop:          make(chan struct{}),
	}
}

func (h *webhook) Serve() {
	for {
		select {
		case c := <-h.issueComments:
			log.Println("New comment received")
			if c.Comment.parsedBody.command == "merge this" {
				log.Println("Going for the merge of PR", c.Issue.Number)
			}

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
			if c.Comment.parsedBody.subject != "" {
				overrideDescr = strings.TrimSpace(c.Comment.parsedBody.subject + "\n\n" + c.Comment.parsedBody.description)
			}

			os.Chdir(c.Repository.FullName)
			res, err := squash(c.Issue.Number, c.Sender.Login, overrideDescr)
			os.Chdir(cur)
			log.Printf("Result:\n%s", res)
			if err != nil {
				log.Println("Failed")
				continue
			}

		case <-h.stop:
			return
		}
	}
}

func (h *webhook) Stop() {
	close(h.stop)
}

func (h *webhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// We only expect POST requests here.
	if r.Method != "POST" {
		http.Error(w, "POST Expected", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If there's a secret set, we except the signature to match.
	if githubSecret != "" {
		// The signature is a SHA1 HMAC of the request body.
		mac := hmac.New(sha1.New, []byte(githubSecret))
		mac.Write(body)
		sig := fmt.Sprintf("sha1=%x", mac.Sum(nil))

		// If it doesn't match the included header, return 401 Unauthorized
		// and abort.
		if hubSig := r.Header.Get("X-Hub-Signature"); hubSig != sig {
			log.Println("Incorrect signature; %s != %s", hubSig, sig)
			http.Error(w, "Incorrect Secret", http.StatusUnauthorized)
			return
		}
	}

	log.Printf("Hook received:\n---\n%s\n---\n\n", body)

	eventType := r.Header.Get("X-Github-Event")
	switch eventType {
	case "issue_comment":
		log.Println("Decoding as a PR comment")
		var c issueComment
		if err := json.Unmarshal(body, &c); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		b := parseBody(c.Comment.Body)
		c.Comment.parsedBody = b
		if b.recipient == h.username {
			h.issueComments <- c
		}

	default:
		log.Printf("Unknown event type %q", eventType)
	}
}
