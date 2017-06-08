package main

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
)

func parseAuthorsFile(r io.Reader) []user {
	var res []user
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		// We assume that neither GitHub username nor email address can
		// contain any spaces.
		var u user
		fields := strings.Fields(sc.Text())
		for i, field := range fields {
			if strings.HasPrefix(field, "(") && strings.HasSuffix(field, ")") {
				u.Name = strings.Join(fields[:i], " ")
				u.Login = field[1 : len(field)-1]
				field = fields[i+1]
				u.Email = field[1 : len(field)-1]
				break
			}
		}
		res = append(res, u)
	}
	return res
}

func getUserFromFile(login, file string) (user, error) {
	fd, err := os.Open(file)
	if err != nil {
		return user{}, err
	}
	defer fd.Close()
	for _, user := range parseAuthorsFile(fd) {
		if user.Login == login {
			return user, nil
		}
	}
	return user{}, errors.New("not found")
}
