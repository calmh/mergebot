package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
)

// The webhook listens on addr for commands to username and send them to the outbox.
type webhook struct {
	addr     string
	secret   string
	username string
	outbox   chan<- comment
	listener net.Listener
}

func newWebhook(outbox chan<- comment, addr, secret, username string) *webhook {
	return &webhook{
		addr:     addr,
		secret:   secret,
		username: username,
		outbox:   outbox,
	}
}

func (h *webhook) Serve() {
	s := http.NewServeMux()
	s.Handle("/", h)

	l, err := net.Listen("tcp", h.addr)
	if err != nil {
		log.Println("Listen:", err)
		return
	}

	log.Println("Web hook receiver listening on", l.Addr())
	h.listener = l
	http.Serve(l, h)
}

func (h *webhook) Stop() {
	h.listener.Close()
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

	// The signature is a SHA1 HMAC of the request body.
	mac := hmac.New(sha1.New, []byte(h.secret))
	mac.Write(body)
	sig := fmt.Sprintf("sha1=%x", mac.Sum(nil))

	// If it doesn't match the included header, return 401 Unauthorized
	// and abort.
	if hubSig := r.Header.Get("X-Hub-Signature"); hubSig != sig {
		log.Printf("Incorrect signature; %s != %s", hubSig, sig)
		http.Error(w, "Incorrect Secret", http.StatusUnauthorized)
		return
	}

	switch eventType := r.Header.Get("X-Github-Event"); eventType {
	case "issue_comment":
		var c comment
		if err := json.Unmarshal(body, &c); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if c.parseBody().recipient == h.username {
			log.Printf("Handling comment by %s on %s", c.Sender.Login, c.Repository.FullName)
			h.outbox <- c
		} else {
			log.Printf("Ignoring comment by %s on %s that does not look like it's for us", c.Sender.Login, c.Repository.FullName)
		}

	default:
		log.Printf("Unknown event type %q, ignored", eventType)
	}
}
