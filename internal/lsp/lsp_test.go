package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestLsp_Start(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		input    []LspMessageKind
		want     []LspMessageKind
		exitCode int
	}{
		{
			"exit",
			[]LspMessageKind{
				requestMessage{
					ID:     intMessageID(1),
					Method: "shutdown",
				},
				notificationMessage{
					Method: "exit",
				},
			},
			[]LspMessageKind{
				responseMessage{
					ID: intMessageID(1),
				},
			},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var message strings.Builder
			for _, i := range tt.input {
				in, err := json.Marshal(i)
				if err != nil {
					t.Fatalf("error on json marshal: %s\n", err.Error())
				}
				if _, err := fmt.Fprintf(
					&message,
					"Content-Length: %d\r\n\r\n%s",
					len(in),
					in,
				); err != nil {
					t.Fatalf("error on Fprintf: %s\n", err.Error())
				}
			}
			reader := strings.NewReader(message.String())
			writer := new(bytes.Buffer)

			l := NewLsp(reader, writer)
			exitCode := l.Start()
			if tt.exitCode != exitCode {
				t.Errorf("Start() = %v, want %v\n", exitCode, tt.exitCode)
			}

			scanner := bufio.NewReader(strings.NewReader(writer.String()))

			t.Log("got", writer.String())

			var got []LspMessageKind
			var hd header

			for {
				s, err := scanner.ReadSlice('\n')
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("error on reading slice %v", err.Error())
				}

				line := string(s)

				switch {
				case strings.HasPrefix(line, "Content-Length:"):
					numstring := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
					l, err := strconv.ParseInt(numstring, 10, 64)
					if err != nil {
						t.Fatalf("could not parse length: %v", err.Error())
					}
					hd.contentLength = l
				case strings.HasPrefix(line, "Content-Type:"):
					ct := strings.TrimSpace(strings.TrimPrefix(line, "Content-Type:"))
					hd.contentType = &ct
				case line == "\r\n":
					var content strings.Builder

					for i := 0; i < int(hd.contentLength); i++ {
						r, err := scanner.ReadByte()
						if err != nil {
							t.Fatalf("error while reading bytes: %v", err.Error())
						}
						content.WriteByte(r)
					}
					t.Log("content", content.String())

					var g lspMessage
					if err := json.Unmarshal([]byte(content.String()), &g); err != nil {
						t.Fatalf("could not unmarshal: %v", err.Error())
					}

					hd.contentLength = 0
					hd.contentType = nil

					got = append(got, g.message)
				}

			}

			opts := cmp.Options{
				cmpopts.IgnoreFields(responseMessage{}, "Message"),
				cmpopts.IgnoreFields(requestMessage{}, "Message"),
				cmpopts.IgnoreFields(notificationMessage{}, "Message"),
			}
			if diff := cmp.Diff(tt.want, got, opts); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
