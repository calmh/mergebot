package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

func reflow(in string, width int) string {
	out := new(bytes.Buffer)  // Reflowed text
	para := new(bytes.Buffer) // Current paragraph
	scanner := bufio.NewScanner(strings.NewReader(in))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || strings.IndexAny(line, " \t") == 0 {
			// Line is empty or starts with space. The previous paragraph has thus ended.
			if para.Len() > 0 {
				fmt.Println("* Flush")
				reflowParagraph(out, para, width)
				para.Reset()
			}

			// Lines like these are passed on verbatim.
			out.WriteString(line)
			out.WriteString("\n")
			continue
		}

		para.WriteString(line)
		para.WriteString(" ")
	}

	if para.Len() > 0 {
		reflowParagraph(out, para, width)
	}

	return strings.TrimRight(out.String(), "\n") + "\n"
}

func reflowParagraph(out io.Writer, in io.Reader, width int) {
	scanner := bufio.NewScanner(in)
	scanner.Split(bufio.ScanWords)
	curWidth := 0
	for scanner.Scan() {
		word := scanner.Text()
		if curWidth+len(word) > width {
			out.Write([]byte("\n"))
			curWidth = 0
		}
		if curWidth > 0 {
			out.Write([]byte(" "))
			curWidth++
		}
		out.Write([]byte(word))
		curWidth += len(word)
	}
	out.Write([]byte("\n\n"))
}
