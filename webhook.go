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
	"sort"
	"strings"
)

type prHandler func(p pr)
type commentHandler func(c comment)

// The webhook listens on addr for commands to username and send them to the outbox.
type webhook struct {
	addr            string
	secret          string
	username        string
	token           string
	commentHandlers map[string]commentHandler
	prHandlers      []prHandler
	listener        net.Listener
}

func newWebhook(addr, secret, username, token string) *webhook {
	return &webhook{
		addr:            addr,
		secret:          secret,
		username:        username,
		token:           token,
		commentHandlers: make(map[string]commentHandler),
	}
}

func (h *webhook) handlePR(fn prHandler) {
	h.prHandlers = append(h.prHandlers, fn)
}

func (h *webhook) handleComment(prefix string, fn commentHandler) {
	h.commentHandlers[prefix] = fn
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

		body := c.parseBody()
		if body.recipient == h.username {
			log.Printf("Handling comment by %s on %s", c.Sender.Login, c.Repository.FullName)
			body.command = strings.ToLower(body.command)
			handled := false
			for prefix, fn := range h.commentHandlers {
				if strings.HasPrefix(body.command, prefix) {
					fn(c)
					handled = true
				}
			}
			if !handled {
				var prefixes []string
				for prefix := range h.commentHandlers {
					prefixes = append(prefixes, prefix)
				}
				sort.Strings(prefixes)

				msg := fmt.Sprintf("I'm sorry, @%s. I'm afraid I don't know what you mean. I know how to `merge` things!", c.Sender.Login)
				c.post(msg, h.username, h.token)
			}
		} else {
			log.Printf("Ignoring comment by %s on %s that does not look like it's for us", c.Sender.Login, c.Repository.FullName)
		}

	case "pull_request":
		var p pr
		if err := json.Unmarshal(body, &p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("Handling pull request %d", p.Number)
		for _, fn := range h.prHandlers {
			fn(p)
		}

	default:
		log.Printf("Unknown event type %q, ignored", eventType)
	}
}
