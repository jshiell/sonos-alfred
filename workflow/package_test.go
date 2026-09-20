package workflow_test

import (
	"archive/zip"
	"bytes"
	"debug/macho"
	"encoding/json"
	"io"
	"os/exec"
	"path/filepath"
	"testing"
)

// packageWorkflow runs the real packaging script and opens the .alfredworkflow it produces.
func packageWorkflow(t *testing.T) *zip.ReadCloser {
	t.Helper()
	out := filepath.Join(t.TempDir(), "Sonos.alfredworkflow")
	if output, err := exec.Command("bash", "../scripts/package.sh", out).CombinedOutput(); err != nil {
		t.Fatalf("package.sh: %v\n%s", err, output)
	}
	archive, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { archive.Close() })
	return archive
}

func file(t *testing.T, archive *zip.ReadCloser, name string) *zip.File {
	t.Helper()
	for _, f := range archive.File {
		if f.Name == name {
			return f
		}
	}
	t.Fatalf("%s is not in the archive", name)
	return nil
}

func contents(t *testing.T, f *zip.File) []byte {
	t.Helper()
	r, err := f.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestPackageHoldsAnExecutableArm64Binary(t *testing.T) {
	binary := file(t, packageWorkflow(t), "sonos-alfred")

	if binary.Mode()&0o111 == 0 {
		t.Errorf("mode = %v, want executable", binary.Mode())
	}
	parsed, err := macho.NewFile(bytes.NewReader(contents(t, binary)))
	if err != nil {
		t.Fatalf("not a Mach-O binary: %v", err)
	}
	if parsed.Cpu != macho.CpuArm64 {
		t.Errorf("cpu = %v, want arm64", parsed.Cpu)
	}
}

// info is the part of info.plist these tests look at.
type info struct {
	Objects []struct {
		Type   string         `json:"type"`
		Config map[string]any `json:"config"`
	} `json:"objects"`
}

func (i info) config(t *testing.T, objectType string) map[string]any {
	t.Helper()
	for _, object := range i.Objects {
		if object.Type == objectType {
			return object.Config
		}
	}
	t.Fatalf("info.plist has no %s", objectType)
	return nil
}

// readInfo reads the archive's info.plist through plutil, which turns any plist format into JSON.
func readInfo(t *testing.T, archive *zip.ReadCloser) info {
	t.Helper()
	convert := exec.Command("plutil", "-convert", "json", "-o", "-", "-")
	convert.Stdin = bytes.NewReader(contents(t, file(t, archive, "info.plist")))
	converted, err := convert.Output()
	if err != nil {
		t.Fatalf("info.plist is not a valid plist: %v", err)
	}
	var parsed info
	if err := json.Unmarshal(converted, &parsed); err != nil {
		t.Fatal(err)
	}
	return parsed
}

func assertConfig(t *testing.T, got map[string]any, want map[string]any) {
	t.Helper()
	for key, wanted := range want {
		if got[key] != wanted {
			t.Errorf("%s = %v, want %v", key, got[key], wanted)
		}
	}
}

func TestScriptFilterOnSonRunsFilterWithTheQueryAndLeavesFilteringToTheBinary(t *testing.T) {
	filter := readInfo(t, packageWorkflow(t)).config(t, "alfred.workflow.input.scriptfilter")

	assertConfig(t, filter, map[string]any{
		"keyword":              "son",
		"type":                 11.0, // an inline script rather than a script file
		"script":               `./sonos-alfred filter "$1"`,
		"scriptargtype":        1.0, // the query arrives as $1
		"alfredfiltersresults": false,
	})
}

func TestChoosingARowRunsDoWithItsAction(t *testing.T) {
	run := readInfo(t, packageWorkflow(t)).config(t, "alfred.workflow.action.script")

	assertConfig(t, run, map[string]any{
		"type":          11.0,
		"script":        `./sonos-alfred do "$1"`,
		"scriptargtype": 1.0,
	})
}
