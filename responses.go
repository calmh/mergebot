package main

import (
	"fmt"
	"time"
)

func noUserResponse(c comment) string {
	return fmt.Sprintf(":question: Couldn't retrieve user information for @%s - not merging.", c.Sender.Login)
}

func thanksResponse(c comment, sha1 string) string {
	return fmt.Sprintf(":ok_hand: Merged as %s. Thanks, @%s!", sha1, c.Issue.User.Login)
}

func waitingResponse(c comment) string {
	return ":clock1: Build status is `pending`. I'll wait to see if this becomes successful and then merge!"
}

func badBuildResponse(c comment, status prState) string {
	return fmt.Sprintf(":no_good: Build status is `%s` -- refusing to merge.", status)
}

func timeoutResponse(c comment, timeout time.Duration) string {
	return fmt.Sprintf(":watch: Patiently waited %v for the build status to turn green, but enough is enough. Let me know when things look good.", timeout)
}

func noAccessResponse(c comment) string {
	return fmt.Sprintf(":hand: I'm sorry, @%s. I'm afraid I can't do that.", c.Sender.Login)
}

func errorResponse(output string) error {
	return fmt.Errorf(":no_entry: Merge failed:\n\n```\n%s\n```\n", output)
}
