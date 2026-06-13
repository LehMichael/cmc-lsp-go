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

type Lsp struct {
	rd io.Reader
	wr io.Writer
}

type header struct {
	contentLength int64
	contentType   *string
}

func NewLsp(rd io.Reader, wr io.Writer) *Lsp {
	return &Lsp{
		rd, wr,
	}
}

func (l *Lsp) Start() int {
	var hd header

	scanner := bufio.NewReader(l.rd)

	for {
		s, err := scanner.ReadSlice('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err.Error())
		}
		line := string(s)
		// fmt.Fprintf(os.Stderr, "csdf?: %d %s\n", utf8.RuneCountInString(line), line)

		switch {
		case strings.HasPrefix(line, "Content-Length:"):
			numstring := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
			l, err := strconv.ParseInt(numstring, 10, 64)
			if err != nil {
				panic(err.Error())
			}
			hd.contentLength = l
		case strings.HasPrefix(line, "Content-Type:"):
			ct := strings.TrimSpace(strings.TrimPrefix(line, "Content-Type:"))
			hd.contentType = &ct
		case line == "\r\n":
			// fmt.Fprintf(
			// 	os.Stderr,
			// 	"header done!\nlength: %v\ntype: %v\n",
			// 	hd.contentLength,
			// 	hd.contentType,
			// )

			var content strings.Builder

			for i := 0; i < int(hd.contentLength); i++ {
				r, err := scanner.ReadByte()
				if err != nil {
					panic(err.Error())
				}
				content.WriteByte(r)
			}

			// fmt.Fprintf(os.Stderr, "asdf?: \n%s\n", content.String())

			if r := l.handleMessage(content.String()); r >= 0 {
				return r
			}

			hd.contentLength = 0
			hd.contentType = nil
		}

	}

	return 1
}

var isShutdown = false

func (l *Lsp) handleMessage(msg string) int {
	var message lspMessage
	if err := json.Unmarshal([]byte(msg), &message); err != nil {
		panic(err.Error())
	}

	if n, ok := message.message.(notificationMessage); ok {
		if isShutdown && n.Method != "exit" {
			return -1
		}
		switch n.Method {
		case "exit":
			if isShutdown {
				return 0
			}
			return 1
		default:
			fmt.Fprintf(os.Stderr, "unknown notification method: %s\n", n.Method)
		}
	} else if r, ok := message.message.(requestMessage); ok {
		if isShutdown && n.Method != "exit" {
			l.respondEmpty(r.ID, &responseError{
				Code: InvalidRequest,
			})
			return -1
		}
		switch r.Method {
		case "shutdown":
			isShutdown = true
			l.respondEmpty(r.ID, nil)
		default:
			fmt.Fprintf(os.Stderr, "unknown request method: %s\n", r.Method)
		}
	} else {
		fmt.Fprintf(os.Stderr, "responseMessage not implemented")
	}

	return -1
}

func (l *Lsp) respondEmpty(id messageID, err *responseError) {
	r := responseMessage{
		Message: Message{
			Jsonrpc: "2.0",
		},
		ID:     id,
		Result: nil,
		Error:  err,
	}

	if jm, err := json.MarshalIndent(r, "", "  "); err != nil {
		panic(err.Error())
	} else {
		fmt.Fprintf(l.wr, "Content-Length: %d\r\n\r\n", len(jm))
		fmt.Fprintln(l.wr, string(jm))
	}
}
