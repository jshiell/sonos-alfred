package sonos_test

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"strings"
	"testing"

	"sonos-alfred/sonos"
)

// recordedExchange is a request the speaker really accepted, captured in the S2 spike.
type recordedExchange struct {
	Action   string `json:"action"`
	Request  string `json:"request"`
	Status   int    `json:"status"`
	Response string `json:"response"`
}

// recorded returns the n-th (0-based) captured exchange for an action.
func recorded(t *testing.T, action string, n int) recordedExchange {
	t.Helper()
	file, err := os.Open("../testdata/s2-dining-room-exchanges.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for scanner.Scan() {
		var exchange recordedExchange
		if err := json.Unmarshal(scanner.Bytes(), &exchange); err != nil {
			t.Fatal(err)
		}
		if exchange.Action == action {
			if n == 0 {
				return exchange
			}
			n--
		}
	}
	t.Fatalf("no recorded %s exchange", action)
	return recordedExchange{}
}

// argsOf decodes the arguments (in order) from a recorded SOAP request.
func argsOf(t *testing.T, request string) []sonos.Arg {
	t.Helper()
	var args []sonos.Arg
	decoder := xml.NewDecoder(strings.NewReader(request))
	depth := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return args
		}
		if err != nil {
			t.Fatal(err)
		}
		switch tok := token.(type) {
		case xml.StartElement:
			depth++
			if depth == 4 {
				args = append(args, sonos.Arg{Name: tok.Name.Local})
			}
		case xml.CharData:
			if depth == 4 {
				args[len(args)-1].Value += string(tok)
			}
		case xml.EndElement:
			depth--
		}
	}
}

// fixtureFile returns the contents of a file in testdata/.
func fixtureFile(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile("../testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
