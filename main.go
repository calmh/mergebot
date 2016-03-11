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
		fmt.Println("Must set Github webhook secret, Github access token, Github user name, and allowed users")
		os.Exit(1)
	}

	allowedUsers := strings.Split(*allow, ",")

	s := newHandler(allowedUsers, *username, *token)
	h := newWebhook(*listenAddr, *secret, *username, *token)
	h.handleComment("merge", s.handleMerge)
	h.handleComment("squash", s.handleMerge)
	h.handlePR(s.handlePullReq)

	main := suture.NewSimple("main")
	main.Add(h)
	main.Serve()
}
