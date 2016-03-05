package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

type comment struct {
	Action string

	Comment struct {
		User struct {
			Login string
		}
		Body string
	}

	Issue struct {
		Number      int
		URL         string
		CommentsURL string `json:"comments_url"`
	}

	Repository struct {
		FullName string `json:"full_name"`
	}

	Sender struct {
		Login string
	}
}

func (c *comment) parseBody() body {
	return parseBody(c.Comment.Body)
}

func (c *comment) post(body, username, token string) {
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(map[string]string{"body": body})
	req, err := http.NewRequest("POST", c.Issue.CommentsURL, buf)
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

func (c *comment) close(username, token string) {
	buf := new(bytes.Buffer)
	json.NewEncoder(buf).Encode(map[string]string{"state": "closed"})
	req, err := http.NewRequest("PATCH", c.Issue.URL, buf)
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
