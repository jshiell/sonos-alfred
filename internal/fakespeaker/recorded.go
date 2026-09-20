package fakespeaker

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

// serviceControlPaths maps a service's URN to where the speaker serves its control endpoint.
var serviceControlPaths = map[string]string{
	"urn:schemas-upnp-org:service:AVTransport:1":           "/MediaRenderer/AVTransport/Control",
	"urn:schemas-upnp-org:service:GroupRenderingControl:1": "/MediaRenderer/GroupRenderingControl/Control",
}

var serviceURN = regexp.MustCompile(`xmlns:u="([^"]+)"`)

// testdataDir is the repository's testdata directory.
func testdataDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata")
}

func testdataFile(t testing.TB, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(testdataDir(), name))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// Recorded returns the n-th (0-based) exchange for an action that a real speaker accepted, captured in the S2 spike.
// Only AVTransport and GroupRenderingControl were captured.
func Recorded(t testing.TB, action string, n int) Exchange {
	t.Helper()
	file, err := os.Open(filepath.Join(testdataDir(), "s2-dining-room-exchanges.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for scanner.Scan() {
		var captured struct {
			Action   string `json:"action"`
			Request  string `json:"request"`
			Status   int    `json:"status"`
			Response string `json:"response"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &captured); err != nil {
			t.Fatal(err)
		}
		if captured.Action != action {
			continue
		}
		if n > 0 {
			n--
			continue
		}
		urn := serviceURN.FindStringSubmatch(captured.Request)[1]
		return Exchange{
			Path:       serviceControlPaths[urn],
			SOAPAction: `"` + urn + "#" + action + `"`,
			Body:       captured.Request,
			Respond:    Response{Status: captured.Status, Body: captured.Response},
		}
	}
	t.Fatalf("no recorded %s exchange", action)
	return Exchange{}
}

// Topology is a GetZoneGroupState call answered with a captured response from testdata/.
func Topology(t testing.TB, responseFixture string) Exchange {
	t.Helper()
	return Exchange{
		Path:       "/ZoneGroupTopology/Control",
		SOAPAction: `"urn:schemas-upnp-org:service:ZoneGroupTopology:1#GetZoneGroupState"`,
		Body:       `<?xml version="1.0" encoding="utf-8"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:GetZoneGroupState xmlns:u="urn:schemas-upnp-org:service:ZoneGroupTopology:1"></u:GetZoneGroupState></s:Body></s:Envelope>`,
		Respond:    Response{Status: http.StatusOK, Body: testdataFile(t, responseFixture)},
	}
}

// Browse is the first page of a ContentDirectory Browse of objectID, answered with a captured response from testdata/.
func Browse(t testing.TB, objectID, responseFixture string) Exchange {
	t.Helper()
	return Exchange{
		Path:       "/MediaServer/ContentDirectory/Control",
		SOAPAction: `"urn:schemas-upnp-org:service:ContentDirectory:1#Browse"`,
		Body: `<?xml version="1.0" encoding="utf-8"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:Browse xmlns:u="urn:schemas-upnp-org:service:ContentDirectory:1"><ObjectID>` +
			objectID + `</ObjectID><BrowseFlag>BrowseDirectChildren</BrowseFlag><Filter>*</Filter><StartingIndex>` + "0" +
			`</StartingIndex><RequestedCount>100</RequestedCount><SortCriteria></SortCriteria></u:Browse></s:Body></s:Envelope>`,
		Respond: Response{Status: http.StatusOK, Body: testdataFile(t, responseFixture)},
	}
}

const envelopeStart = `<?xml version="1.0" encoding="utf-8"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body>`

// AVTransport is an AVTransport action with the given argument XML, answered with an empty response. Use it where no
// exchange was recorded; the request shape comes from the svrooij AVTransport docs.
func AVTransport(action, argsXML string) Exchange {
	const urn = "urn:schemas-upnp-org:service:AVTransport:1"
	return Exchange{
		Path:       serviceControlPaths[urn],
		SOAPAction: `"` + urn + "#" + action + `"`,
		Body:       envelopeStart + `<u:` + action + ` xmlns:u="` + urn + `">` + argsXML + `</u:` + action + `></s:Body></s:Envelope>`,
		Respond: Response{Status: http.StatusOK, Body: `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body><u:` +
			action + `Response xmlns:u="` + urn + `"></u:` + action + `Response></s:Body></s:Envelope>`},
	}
}
