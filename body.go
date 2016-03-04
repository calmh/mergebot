package main

import (
	"regexp"
	"strings"
)

type body struct {
	recipient   string
	command     string
	subject     string
	description string
}

var rcptExp = regexp.MustCompile(`@([^\s:]+):`)

func parseBody(s string) body {
	var res body

	lines := strings.Split(s, "\n")
	res.command = lines[0]

	if m := rcptExp.FindStringSubmatch(lines[0]); len(m) > 0 {
		res.recipient = m[1]
		res.command = strings.Join(strings.Fields(lines[0])[1:], " ")
	}

	if len(lines) > 1 {
		lines = lines[1:] // Skip command
		for strings.TrimSpace(lines[0]) == "" {
			// Skip blank lines after command
			lines = lines[1:]
		}
		res.subject = lines[0]
	}

	if len(lines) > 1 {
		lines = lines[1:] // Skip subject
		for strings.TrimSpace(lines[0]) == "" {
			// Skip blank lines after subject
			lines = lines[1:]
		}
		res.description = strings.Join(lines, "\n")
	}

	return res
}
