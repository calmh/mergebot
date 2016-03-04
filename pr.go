package main

type issueComment struct {
	Action  string
	Comment struct {
		User struct {
			Login string
		}
		Body       string
		parsedBody body
	}
	Issue struct {
		Number int
		URL    string
	}
	Repository struct {
		FullName string `json:"full_name"`
		PullsURL string `json:"pulls_url"`
	}
	Sender struct {
		Login string
	}
}
