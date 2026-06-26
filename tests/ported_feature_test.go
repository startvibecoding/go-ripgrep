package tests

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
)

// Ported from: f20_no_filename in feature.rs
func TestFeature20NoFilename(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		dir.Create("sherlock", SHERLOCK)
		cmd.Args("--no-filename", "Sherlock")

		expected := "For the Doctor Watsons of this world, as opposed to the Sherlock\n" +
			"be, to a very large extent, the result of luck. Sherlock Holmes\n"

		Equnice(t, expected, cmd.Stdout())
	})
}

// Ported from: f34_only_matching in feature.rs
func TestFeature34OnlyMatching(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		dir.Create("sherlock", SHERLOCK)
		cmd.Args("-o", "Sherlock")

		expected := "sherlock:Sherlock\n" +
			"sherlock:Sherlock\n"

		Equnice(t, expected, cmd.Stdout())
	})
}

// Ported from: f34_only_matching_no_filename in feature.rs
func TestFeature34OnlyMatchingNoFilename(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		dir.Create("sherlock", SHERLOCK)
		cmd.Args("-o", "--no-filename", "Sherlock", "sherlock")

		expected := "Sherlock\n" +
			"Sherlock\n"

		Equnice(t, expected, cmd.Stdout())
	})
}

// Ported from: stdin search
func TestFeatureStdinPipe(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		cmd.Arg("Sherlock")
		expected := "For the Doctor Watsons of this world, as opposed to the Sherlock\n" +
			"be, to a very large extent, the result of luck. Sherlock Holmes\n"

		Equnice(t, expected, cmd.Pipe([]byte(SHERLOCK)))
	})
}

func TestFeatureReplaceSimple(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		dir.Create("sherlock", SHERLOCK)
		cmd.Args("-r", "Holmes", "Sherlock")

		expected := "sherlock:For the Doctor Watsons of this world, as opposed to the Holmes\n" +
			"sherlock:be, to a very large extent, the result of luck. Holmes Holmes\n"

		Equnice(t, expected, cmd.Stdout())
	})
}

func TestFeatureReplaceCaptureGroups(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		dir.Create("sherlock", SHERLOCK)
		cmd.Args("-r", "Dr. $1", `(\w+) Watson`)

		expected := "sherlock:For the Dr. Doctors of this world, as opposed to the Sherlock\n" +
			"sherlock:but Dr. Doctor has to have it taken out for him and dusted,\n"

		Equnice(t, expected, cmd.Stdout())
	})
}

func TestFeatureReplaceOnlyMatching(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		dir.Create("sherlock", SHERLOCK)
		cmd.Args("-o", "-r", "Mycroft", "Sherlock")

		expected := "sherlock:Mycroft\n" +
			"sherlock:Mycroft\n"

		Equnice(t, expected, cmd.Stdout())
	})
}

func TestFeatureTypesInclude(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		dir.Create("a.go", `func main() { println("golang") }`)
		dir.Create("b.rs", `fn main() { println!("rust"); }`)
		cmd.Args("-t", "go", "main")

		expected := `a.go:func main() { println("golang") }` + "\n"
		Equnice(t, expected, cmd.Stdout())
	})
}

func TestFeatureTypesExclude(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		dir.Create("a.go", `func main() { println("golang") }`)
		dir.Create("b.rs", `fn main() { println!("rust"); }`)
		cmd.Args("-T", "rust", "main")

		expected := `a.go:func main() { println("golang") }` + "\n"
		Equnice(t, expected, cmd.Stdout())
	})
}

func TestFeatureTypeList(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		cmd.Args("--type-list")
		output := cmd.Stdout()
		if !strings.Contains(output, "go: *.go") {
			t.Errorf("type list missing go: *.go. Output: %s", output)
		}
		if !strings.Contains(output, "rust: *.rs") {
			t.Errorf("type list missing rust: *.rs. Output: %s", output)
		}
	})
}

func TestFeatureSortPath(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		// Create files such that alphabetical order is clear
		dir.Create("z.txt", "match")
		dir.Create("y.txt", "match")
		dir.Create("x.txt", "match")

		cmd.Args("--sort", "path", "match")

		expected := "x.txt:match\n" +
			"y.txt:match\n" +
			"z.txt:match\n"

		Equnice(t, expected, cmd.Stdout())
	})
}

func TestFeatureSortReversePath(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		dir.Create("z.txt", "match")
		dir.Create("y.txt", "match")
		dir.Create("x.txt", "match")

		cmd.Args("--sortr", "path", "match")

		expected := "z.txt:match\n" +
			"y.txt:match\n" +
			"x.txt:match\n"

		Equnice(t, expected, cmd.Stdout())
	})
}

func TestFeatureDecompressionGzip(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		_, _ = gw.Write([]byte("hello world from gzip\n"))
		_ = gw.Close()

		dir.CreateBytes("test.gz", buf.Bytes())

		cmd.Args("-z", "world")
		expected := "test.gz:hello world from gzip\n"
		Equnice(t, expected, cmd.Stdout())
	})
}

func TestFeatureDecompressionZip(t *testing.T) {
	RunRgTest(t, func(dir *Dir, cmd *TestCommand) {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)

		f1, err := zw.Create("hello.txt")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f1.Write([]byte("hello world from zip\n"))

		f2, err := zw.Create("sub/world.txt")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f2.Write([]byte("another matching line\n"))

		_ = zw.Close()

		dir.CreateBytes("test.zip", buf.Bytes())

		cmd.Args("-z", "world|matching")

		expected := "test.zip//hello.txt:hello world from zip\n" +
			"test.zip//sub/world.txt:another matching line\n"

		Equnice(t, expected, cmd.Stdout())
	})
}
