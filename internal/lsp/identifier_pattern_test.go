package lsp

import (
	"bytes"
	"strings"
	"testing"
)

func TestSegmentPatternsOverlap(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  bool
	}{
		{left: `ChanGeoAxNo$(Up.AxNo)`, right: "ChanGeoAxNo1", want: true},
		{left: `ChanGeoAxNo$(Up.AxNo)`, right: `Chan$(Up.a)No1`, want: true},
		{left: `A$(Up.x)`, right: `$(Up.y)B`, want: true},
		{left: `Prefix$(Up.x)A`, right: `Prefix$(Up.y)B`, want: false},
		{left: "ChanGeoAxNo1", right: "changeoaxno1", want: true},
		{left: "ChanGeoAxNo1", right: "ChanGeoAxNo2", want: false},
	}
	for _, test := range tests {
		t.Run(test.left+"_"+test.right, func(t *testing.T) {
			if got := segmentPatternsOverlap(patternFromRaw(test.left), patternFromRaw(test.right)); got != test.want {
				t.Fatalf("overlap(%q, %q) = %v, want %v", test.left, test.right, got, test.want)
			}
		})
	}
}

func TestReferencesIncludeCompatibleDynamicIdentifiers(t *testing.T) {
	const uri = "file:///tmp/dynamic-references.upscr"
	const source = `Up.ChanGeoAxNo$(Up.AxNo) = 1
Up.ChanGeoAxNo1 = 2
Up.Chan$(Up.a)No1 = 3
Up.Unrelated = 4
Up.AxNo = 1
`
	lines := strings.Split(source, "\n")
	var output bytes.Buffer
	server := NewLsp(bytes.NewReader(nil), &output)
	if err := server.overlay.Open(uri, source, 1); err != nil {
		t.Fatal(err)
	}

	server.handleReferences(requestMessage{ID: intMessageID(1), Params: marshalParams(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position": map[string]any{
			"line": 0, "character": strings.Index(lines[0], "ChanGeoAxNo") + 2,
		},
	})})
	var outerResponse struct {
		Result []location `json:"result"`
	}
	readResponse(t, &output, &outerResponse)
	assertReferenceLines(t, outerResponse.Result, []int{0, 1, 2})

	server.handleReferences(requestMessage{ID: intMessageID(2), Params: marshalParams(t, map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position": map[string]any{
			"line": 0, "character": strings.LastIndex(lines[0], "AxNo") + 1,
		},
	})})
	var replacementResponse struct {
		Result []location `json:"result"`
	}
	readResponse(t, &output, &replacementResponse)
	assertReferenceLines(t, replacementResponse.Result, []int{0, 4})
}

func assertReferenceLines(t *testing.T, locations []location, want []int) {
	t.Helper()
	if len(locations) != len(want) {
		t.Fatalf("reference locations = %#v, want lines %v", locations, want)
	}
	for index, line := range want {
		if locations[index].Range.Start.Line != line {
			t.Fatalf("reference %d starts on line %d, want %d: %#v", index, locations[index].Range.Start.Line, line, locations)
		}
	}
}
