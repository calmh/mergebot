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
	branches := flag.Bool("branches", false, "Keep and update branches for PRs")
	dbfile := flag.String("dbfile", "mergebot.db", "Database file")
	flag.Parse()

	if *secret == "" || *token == "" || *username == "" {
		fmt.Println("Must set Github webhook secret, Github access token, and Github user name")
		os.Exit(1)
	}

	allowedUsers := strings.Split(*allow, ",")

	db, err := OpenDB(*dbfile)
	if err != nil {
		fmt.Println("Opening database:", err)
		os.Exit(1)
	}

	s := newHandler(allowedUsers, *username, *token, *branches, db)
	h := newWebhook(*listenAddr, *secret, *username, *token)
	h.handleComment("merge", s.handleMerge)
	h.handleComment("squash", s.handleMerge)
	h.handleComment("stop", s.handleStop)
	h.handleComment("don't", s.handleStop)
	h.handleComment("prevent", s.handleStop)
	h.handleComment("lgtm", s.handleLGTM)
	h.handlePR(s.handlePullReq)

	main := suture.NewSimple("main")
	main.Add(h)
	main.Serve()
}
