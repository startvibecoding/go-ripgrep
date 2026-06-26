package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const SHERLOCK = "For the Doctor Watsons of this world, as opposed to the Sherlock\n" +
	"Holmeses, success in the province of detective work must always\n" +
	"be, to a very large extent, the result of luck. Sherlock Holmes\n" +
	"can extract a clew from a wisp of straw or a flake of cigar ash;\n" +
	"but Doctor Watson has to have it taken out for him and dusted,\n" +
	"and exhibited clearly, with a label attached.\n"

const SHERLOCK_CRLF = "For the Doctor Watsons of this world, as opposed to the Sherlock\r\n" +
	"Holmeses, success in the province of detective work must always\r\n" +
	"be, to a very large extent, the result of luck. Sherlock Holmes\r\n" +
	"can extract a clew from a wisp of straw or a flake of cigar ash;\r\n" +
	"but Doctor Watson has to have it taken out for him and dusted,\r\n" +
	"and exhibited clearly, with a label attached.\r\n"

type Dir struct {
	t    *testing.T
	Path string
}

func (d *Dir) Create(name, content string) {
	fullPath := filepath.Join(d.Path, name)
	err := os.MkdirAll(filepath.Dir(fullPath), 0755)
	if err != nil {
		d.t.Fatalf("failed to create directory for %s: %v", name, err)
	}
	err = os.WriteFile(fullPath, []byte(content), 0644)
	if err != nil {
		d.t.Fatalf("failed to create file %s: %v", name, err)
	}
}

func (d *Dir) CreateBytes(name string, content []byte) {
	fullPath := filepath.Join(d.Path, name)
	err := os.MkdirAll(filepath.Dir(fullPath), 0755)
	if err != nil {
		d.t.Fatalf("failed to create directory for %s: %v", name, err)
	}
	err = os.WriteFile(fullPath, content, 0644)
	if err != nil {
		d.t.Fatalf("failed to create file %s: %v", name, err)
	}
}

type TestCommand struct {
	t      *testing.T
	Binary string
	Dir    string
	args   []string
	stdin  []byte
}

func (c *TestCommand) Arg(arg string) *TestCommand {
	c.args = append(c.args, arg)
	return c
}

func (c *TestCommand) Args(args ...string) *TestCommand {
	c.args = append(c.args, args...)
	return c
}

func (c *TestCommand) Pipe(stdin []byte) string {
	c.stdin = stdin
	return c.Stdout()
}

func (c *TestCommand) Stdout() string {
	cmd := exec.Command(c.Binary, c.args...)
	cmd.Dir = c.Dir
	if len(c.stdin) > 0 {
		cmd.Stdin = bytes.NewReader(c.stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run() // Exit codes 0, 1 are expected.
	return stdout.String()
}

func (c *TestCommand) Stderr() string {
	cmd := exec.Command(c.Binary, c.args...)
	cmd.Dir = c.Dir
	if len(c.stdin) > 0 {
		cmd.Stdin = bytes.NewReader(c.stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run()
	return stderr.String()
}

func RunRgTest(t *testing.T, f func(dir *Dir, cmd *TestCommand)) {
	// Build/resolve binary
	binaryPath, err := filepath.Abs("../bin/rg")
	if err != nil {
		t.Fatalf("failed to resolve binary path: %v", err)
	}

	// Make sure the binary is built (or rebuild if needed)
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/rg")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	// Create temp directory for the test
	tmpDir, err := os.MkdirTemp("", "rgtest-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dir := &Dir{t: t, Path: tmpDir}
	cmd := &TestCommand{t: t, Binary: binaryPath, Dir: tmpDir}

	f(dir, cmd)
}

// Equnice is a helper to assert stdout matches expected string with normalized newlines
func Equnice(t *testing.T, expected, actual string) {
	expectedNorm := strings.ReplaceAll(expected, "\r\n", "\n")
	actualNorm := strings.ReplaceAll(actual, "\r\n", "\n")
	if expectedNorm != actualNorm {
		t.Errorf("Mismatch!\nExpected:\n%s\nActual:\n%s", expectedNorm, actualNorm)
	}
}
