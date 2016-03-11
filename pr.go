package main

type pr struct {
	Action     string
	Number     int
	Repository struct {
		FullName string `json:"full_name"`
	}
}
