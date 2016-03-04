package main

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type script struct {
	output *bytes.Buffer
	err    error
}

func newScript() *script {
	return &script{
		output: new(bytes.Buffer),
	}
}

func (s *script) Error() error {
	return s.err
}

func (s *script) run(bin string, args ...string) string {
	if s.err != nil {
		return ""
	}

	cmd := exec.Command(bin, args...)
	return s.runCmd(cmd)
}

func (s *script) runPipe(stdin io.Reader, bin string, args ...string) string {
	if s.err != nil {
		return ""
	}

	cmd := exec.Command(bin, args...)
	cmd.Stdin = stdin
	return s.runCmd(cmd)
}

func (s *script) runCmd(cmd *exec.Cmd) string {
	cmdLine := new(bytes.Buffer)
	for i, arg := range cmd.Args {
		if i > 0 {
			cmdLine.WriteRune(' ')
		}
		if strings.ContainsAny(arg, " \t\r\n") {
			fmt.Fprintf(cmdLine, "%q", arg)
		} else {
			cmdLine.WriteString(arg)
		}
	}
	fmt.Fprintln(s.output, "$", cmdLine.String())

	bs, err := cmd.CombinedOutput()
	if err != nil {
		s.err = err
	}
	s.output.Write(bs)

	out := strings.TrimRight(string(bs), " \r\n\t")
	return out
}
