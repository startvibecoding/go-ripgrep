package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationCLI(t *testing.T) {
	// 1. Build the binary
	binaryPath, err := filepath.Abs("../rg")
	if err != nil {
		t.Fatalf("failed to resolve binary path: %v", err)
	}

	// Make sure the binary is built
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/rg")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}

	// 2. Create a temporary test environment
	tmpDir, err := os.MkdirTemp("", "rg-integration-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	file1 := filepath.Join(tmpDir, "hello.txt")
	if err := os.WriteFile(file1, []byte("Hello World!\nWelcome to go-ripgrep.\nThis is awesome.\n"), 0644); err != nil {
		t.Fatalf("failed to write hello.txt: %v", err)
	}

	file2 := filepath.Join(tmpDir, "secret.tmp")
	if err := os.WriteFile(file2, []byte("password=123456\nHello agent.\n"), 0644); err != nil {
		t.Fatalf("failed to write secret.tmp: %v", err)
	}

	// Create gitignore
	gitignore := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignore, []byte("*.tmp\n"), 0644); err != nil {
		t.Fatalf("failed to write .gitignore: %v", err)
	}

	// Test Case 1: Simple match (case-sensitive)
	{
		cmd := exec.Command(binaryPath, "Welcome", tmpDir)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("Test Case 1 failed: error: %v", err)
		}
		if !strings.Contains(out.String(), "Welcome to go-ripgrep.") {
			t.Errorf("Test Case 1 failed: expected match not found. Output: %q", out.String())
		}
	}

	// Test Case 2: Case-insensitive match (-i)
	{
		cmd := exec.Command(binaryPath, "-i", "welcome", tmpDir)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("Test Case 2 failed: error: %v", err)
		}
		if !strings.Contains(out.String(), "Welcome to go-ripgrep.") {
			t.Errorf("Test Case 2 failed: expected match not found. Output: %q", out.String())
		}
	}

	// Test Case 3: Respecting gitignore (by default secret.tmp is ignored)
	{
		cmd := exec.Command(binaryPath, "agent", tmpDir)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		// Should exit with status code 1 (no matches found)
		if err == nil {
			t.Errorf("Test Case 3 failed: expected exit status 1 due to gitignore, got 0. Output: %q", out.String())
		}
	}

	// Test Case 4: Overriding gitignore (--no-ignore)
	{
		cmd := exec.Command(binaryPath, "--no-ignore", "agent", tmpDir)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("Test Case 4 failed: error: %v", err)
		}
		if !strings.Contains(out.String(), "Hello agent.") {
			t.Errorf("Test Case 4 failed: expected match in ignored file when --no-ignore active. Output: %q", out.String())
		}
	}

	// Test Case 5: Fixed strings (-F)
	{
		cmd := exec.Command(binaryPath, "-F", "World!", tmpDir)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("Test Case 5 failed: error: %v", err)
		}
		if !strings.Contains(out.String(), "Hello World!") {
			t.Errorf("Test Case 5 failed: expected match. Output: %q", out.String())
		}
	}

	// Test Case 6: Context lines (-C 1)
	{
		cmd := exec.Command(binaryPath, "-C", "1", "Welcome", tmpDir)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("Test Case 6 failed: error: %v", err)
		}
		expectedLines := []string{"Hello World!", "Welcome to go-ripgrep.", "This is awesome."}
		for _, el := range expectedLines {
			if !strings.Contains(out.String(), el) {
				t.Errorf("Test Case 6 failed: expected line %q not found. Output: %q", el, out.String())
			}
		}
	}

	// Test Case 7: Word Match (-w)
	{
		cmd := exec.Command(binaryPath, "-w", "rip", tmpDir)
		err := cmd.Run()
		if err == nil {
			t.Errorf("Test Case 7 failed: expected no match for substring 'rip' with word-regexp active")
		}

		cmd2 := exec.Command(binaryPath, "-w", "awesome", tmpDir)
		if err := cmd2.Run(); err != nil {
			t.Errorf("Test Case 7 failed: expected match for whole word 'awesome'")
		}
	}

	// Test Case 8: JSON Output (--json)
	{
		cmd := exec.Command(binaryPath, "--json", "Welcome", tmpDir)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("Test Case 8 failed: error: %v", err)
		}
		output := out.String()
		if !strings.Contains(output, `"type":"begin"`) || !strings.Contains(output, `"type":"match"`) || !strings.Contains(output, `"type":"end"`) {
			t.Errorf("Test Case 8 failed: output was not valid NDJSON. Output: %q", output)
		}
	}

	// Test Case 9: Stdin piping
	{
		cmd := exec.Command(binaryPath, "golang")
		var in bytes.Buffer
		var out bytes.Buffer
		in.WriteString("go-ripgrep is written in golang\nand it is fast!\n")
		cmd.Stdin = &in
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("Test Case 9 failed: error: %v", err)
		}
		if !strings.Contains(out.String(), "go-ripgrep is written in golang") {
			t.Errorf("Test Case 9 failed: expected match from stdin pipe. Output: %q", out.String())
		}
	}

	// Test Case 10: Case sensitivity override (-i -s vs -s -i)
	{
		// -i -s should make it case-sensitive (capital W only)
		cmd1 := exec.Command(binaryPath, "-i", "-s", "welcome", tmpDir)
		err := cmd1.Run()
		if err == nil {
			t.Errorf("Test Case 10 failed: expected -i -s to be case-sensitive (no match for 'welcome')")
		}

		// -s -i should make it case-insensitive (match 'welcome')
		cmd2 := exec.Command(binaryPath, "-s", "-i", "welcome", tmpDir)
		var out2 bytes.Buffer
		cmd2.Stdout = &out2
		if err := cmd2.Run(); err != nil {
			t.Errorf("Test Case 10 failed: expected -s -i to match 'welcome'")
		}
	}

	// Test Case 11: Line number suppression (-N)
	{
		cmd := exec.Command(binaryPath, "-N", "Welcome", tmpDir)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("Test Case 11 failed: error: %v", err)
		}
		output := out.String()
		if strings.Contains(output, "2:") {
			t.Errorf("Test Case 11 failed: output contains line number but -N was active. Output: %q", output)
		}
	}

	// Test Case 12: Filename suppression (-I)
	{
		cmd := exec.Command(binaryPath, "-I", "Welcome", tmpDir)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("Test Case 12 failed: error: %v", err)
		}
		output := out.String()
		if strings.Contains(output, "hello.txt") {
			t.Errorf("Test Case 12 failed: output contains filename but -I was active. Output: %q", output)
		}
	}

	// Test Case 13: Only matching (-o)
	{
		cmd := exec.Command(binaryPath, "-o", "awesome", tmpDir)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("Test Case 13 failed: error: %v", err)
		}
		output := out.String()
		// Output should contain only "awesome", not "This is awesome."
		if !strings.Contains(output, "awesome") || strings.Contains(output, "This is") {
			t.Errorf("Test Case 13 failed: expected only matching text. Output: %q", output)
		}
	}

	// Test Case 14: Count matches (-c)
	{
		cmd := exec.Command(binaryPath, "-c", "Welcome", tmpDir)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("Test Case 14 failed: error: %v", err)
		}
		output := out.String()
		// Output should end with ":1" indicating 1 match in hello.txt
		if !strings.Contains(output, "hello.txt:1") {
			t.Errorf("Test Case 14 failed: expected count output like hello.txt:1. Output: %q", output)
		}
	}

	// Test Case 15: Quiet mode (-q)
	{
		// When match exists, should exit 0 and output nothing
		cmd := exec.Command(binaryPath, "-q", "Welcome", tmpDir)
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Errorf("Test Case 15 failed: expected quiet mode to exit with status 0, got error: %v", err)
		}
		if out.Len() > 0 {
			t.Errorf("Test Case 15 failed: quiet mode should output nothing but got: %q", out.String())
		}

		// When match does not exist, should exit 1 and output nothing
		cmd2 := exec.Command(binaryPath, "-q", "nonexistent_pattern", tmpDir)
		var out2 bytes.Buffer
		cmd2.Stdout = &out2
		err := cmd2.Run()
		if err == nil {
			t.Errorf("Test Case 15 failed: expected quiet mode to exit with non-zero when no matches")
		}
		if out2.Len() > 0 {
			t.Errorf("Test Case 15 failed: quiet mode should output nothing on no match but got: %q", out2.String())
		}
	}
}
