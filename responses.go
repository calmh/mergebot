package main

import (
	"fmt"
	"time"
)

func noUserResponse(c comment) string {
	return fmt.Sprintf("@%s: Couldn't retrieve your user information - not merging.", c.Sender.Login)
}

func thanksResponse(c comment, sha1 string) string {
	return fmt.Sprintf(":ok_hand: Merged as %s. Thanks, @%s!", sha1, c.Issue.User.Login)
}

func waitingResponse(c comment) string {
	return fmt.Sprintf("@%s: Build status is `pending`. I'll wait until it goes green and then merge!", c.Sender.Login)
}

func badBuildResponse(c comment, status prState) string {
	return fmt.Sprintf("@%s: Build status is `%s` -- refusing to merge.", c.Sender.Login, status)
}

func timeoutResponse(c comment, timeout time.Duration) string {
	return fmt.Sprintf("@%s: Patiently waited %v for the build status to turn green, but enough is enough.", c.Sender.Login, timeout)
}

func noAccessResponse(c comment) string {
	return fmt.Sprintf(":hand: I'm sorry, @%s. I'm afraid I can't do that.", c.Sender.Login)
}

func errorResponse(c comment, output string) string {
	return fmt.Sprintf("@%s: Merge failed:\n\n```\n%s\n```\n", c.Sender.Login, output)
}

func cloneFailedResponse(c comment, output string) string {
	return fmt.Sprintf("@%s: Clone failed:\n\n```\n%s\n```\n", c.Sender.Login, output)
}

func notMergingResponse(c comment) string {
	return fmt.Sprintf("@%s: Preventing merge for the time being. Push a new revision to reset!", c.Sender.Login)
}

func alreadyPendingResponse(c comment) string {
	return fmt.Sprintf("@%s: There's already a merge pending for this PR.", c.Sender.Login)
}

func lgtmResponse(c comment) string {
	return fmt.Sprintf("@%s: Noted! Need another LGTM or explicit merge command.", c.Sender.Login)
}

func tcErrorResponse(c comment, err error) string {
	return fmt.Sprintf("@%s: Triggered the build, but TC said `%v`. Maybe it worked, maybe it didn't.", c.Sender.Login, err)
}
