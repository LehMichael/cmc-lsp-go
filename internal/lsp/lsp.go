package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Lsp struct{}

type header struct {
	contentLength int64
	contentType   *string
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

func NewLsp() {
	fmt.Println("Hello, World!")

	var hd header
	headerDone := false

	stdin := countingReader{
		r: os.Stdin,
		n: 0,
	}

	contentStart := stdin.n
	var content strings.Builder

	scanner := bufio.NewScanner(&stdin)

	for scanner.Scan() {
		s := scanner.Text()

		if headerDone {
			content.WriteString(s)
			content.WriteString("\n")

			if stdin.n-contentStart >= hd.contentLength {
				fmt.Printf("got: %v expected: %v\n", stdin.n-contentStart, hd.contentLength)
				fmt.Printf("aaaaa:\n\n%v\n\n\n", content.String())
				var message RequestMessage
				if err := json.Unmarshal([]byte(content.String()), &message); err != nil {
					panic(err.Error())
				}
				if jm, err := json.MarshalIndent(message, "", "  "); err != nil {
					panic(err.Error())
				} else {
					fmt.Printf("ffffff: \n%v\n", string(jm))
				}

			}
		}

		switch {
		case strings.HasPrefix(s, "Content-Length:"):
			numstring := strings.TrimSpace(string([]rune(s)[15:]))
			// fmt.Printf("aaasdf: %v\n", numstring)
			l, e := strconv.ParseInt(numstring, 10, 64)
			if e != nil {
				panic(e.Error())
			}
			hd.contentLength = l
			// fmt.Printf("len: %v\n", l)
		case strings.HasPrefix(s, "Content-Type:"):
			ct := strings.TrimSpace(string([]rune(s)[13:]))
			hd.contentType = &ct
		case s == "":
			headerDone = true
			contentStart = stdin.n
			fmt.Printf(
				"header done!\nlength: %v\ntype: %v\n",
				hd.contentLength,
				hd.contentType,
			)
		}

	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading input:", err)
	}
}
