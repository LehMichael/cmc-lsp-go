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

func NewLsp() int {
	var hd header
	headerDone := false

	stdin := countingReader{
		r: os.Stdin,
		n: 0,
	}

	var content strings.Builder

	scanner := bufio.NewScanner(&stdin)

	for scanner.Scan() {
		s := scanner.Text()

		if headerDone {
			content.WriteString(s)
			content.WriteString("\n")

			if stdin.n >= hd.contentLength {
				if r := handleMessage(content.String()); r >= 0 {
					return r
				}
				headerDone = false
				content.Reset()
			}
		}

		switch {
		case strings.HasPrefix(s, "Content-Length:"):
			numstring := strings.TrimSpace(string([]rune(s)[15:]))
			l, e := strconv.ParseInt(numstring, 10, 64)
			if e != nil {
				panic(e.Error())
			}
			hd.contentLength = l
		case strings.HasPrefix(s, "Content-Type:"):
			ct := strings.TrimSpace(string([]rune(s)[13:]))
			hd.contentType = &ct
		case s == "":
			headerDone = true
			stdin.n = 0
			fmt.Fprintf(
				os.Stderr,
				"header done!\nlength: %v\ntype: %v\n",
				hd.contentLength,
				hd.contentType,
			)
		}

	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
	}

	return 1
}

var isShutdown = false

func handleMessage(msg string) int {
	var message requestMessage
	if err := json.Unmarshal([]byte(msg), &message); err != nil {
		panic(err.Error())
	}

	// if jm, err := json.MarshalIndent(message, "", "  "); err != nil {
	// 	panic(err.Error())
	// } else {
	// 	fmt.Fprintf(os.Stderr, "ffffff: \n%v\n", string(jm))
	// }

	if isShutdown && message.Method != "exit" {
		respondEmpty(message.ID, &responseError{
			Code: InvalidRequest,
		})
		return -1
	}

	switch message.Method {
	case "shutdown":
		isShutdown = true
		respondEmpty(message.ID, nil)
	case "exit":
		if isShutdown {
			return 0
		}
		return 1
	}

	return -1
}

func respondEmpty(id messageID, err *responseError) {
	r := responseMessage{
		message: message{
			Jsonrpc: "2.0",
		},
		ID:     id,
		Result: nil,
		Error:  err,
	}

	if jm, err := json.MarshalIndent(r, "", "  "); err != nil {
		panic(err.Error())
	} else {
		fmt.Println(string(jm))
	}
}
