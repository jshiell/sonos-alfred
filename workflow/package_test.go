package workflow_test

import (
	"archive/zip"
	"bytes"
	"debug/macho"
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
