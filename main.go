package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/thejerf/suture"
)

func main() {
	listenAddr := flag.String("listen", ":8080", "Listen address")
	secret := flag.String("secret", "", "Github webhook secret")
	token := flag.String("token", "", "Github access token")
	username := flag.String("username", "", "Github user name")
	allow := flag.String("allow", "", "Comma separeted list of allowed maintainers")
	flag.Parse()

	if *secret == "" || *token == "" || *username == "" || *allow == "" {
		fmt.Println("Must set Github webhook secret, Github access token, and Github user name")
		os.Exit(1)
	}

	allowedUsers := strings.Split(*allow, ",")

	main := suture.NewSimple("main")
	comments := make(chan comment)

	h := newWebhook(comments, *listenAddr, *secret, *username)
	main.Add(h)

	s := newSquasher(comments, allowedUsers, *username, *token)
	main.Add(s)

	main.Serve()
}
