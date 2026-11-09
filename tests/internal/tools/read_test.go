package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
)

func TestReadFile(t *testing.T) {
	// Define the generated directory path
	genDir := filepath.Join("generated", "read")
	err := os.MkdirAll(genDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	originalWd, _ := os.Getwd()
	os.Chdir(genDir)
	defer os.Chdir(originalWd)

	fileName := "test.txt"
	content := "hello world"

	err = os.WriteFile(fileName, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		workingDir string
		filePath   string
		want       string
	}{
		{
			name:       "Success read",
			workingDir: ".",
			filePath:   fileName,
			want:       "hello world",
		},
		{
			name:       "Path outside",
			workingDir: ".",
			filePath:   "../outside.txt",
			want:       "Error: Filepath outside the permitted working directory",
		},
		{
			name:       "File not found",
			workingDir: ".",
			filePath:   "nonexistent.txt",
			want:       filetools.READ_NOT_FOUND_NOTICE + "nonexistent.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filetools.ReadFile(tt.workingDir, tt.filePath)
			if tt.name == "Success read" {
				if got != tt.want {
					t.Errorf("ReadFile() = %q, want %q", got, tt.want)
				}
			} else {
				if !strings.HasPrefix(got, "Error") {
					t.Errorf("ReadFile() = %q, want error message starting with 'Error'", got)
				}
				if got != tt.want {
					t.Errorf("ReadFile() = %q, want %q", got, tt.want)
				}
			}
		})
	}
}

// TestReadFileTruncatesAtCap pins the read cap: a file of exactly
// READ_MAX_CHARS_TO_READ bytes comes back whole, one byte more comes back cut
// at the cap with the truncation marker.
func TestReadFileTruncatesAtCap(t *testing.T) {
	genDir := filepath.Join("generated", "read")
	if err := os.MkdirAll(genDir, 0755); err != nil {
		t.Fatal(err)
	}

	originalWd, _ := os.Getwd()
	os.Chdir(genDir)
	defer os.Chdir(originalWd)

	atCap := strings.Repeat("a", filetools.READ_MAX_CHARS_TO_READ)
	if err := os.WriteFile("at-cap.txt", []byte(atCap), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("over-cap.txt", []byte(atCap+"b"), 0644); err != nil {
		t.Fatal(err)
	}

	if got := filetools.ReadFile(".", "at-cap.txt"); got != atCap {
		t.Errorf("ReadFile(at cap) returned %d bytes, want the whole %d", len(got), len(atCap))
	}
	want := atCap + "\n... (conteúdo truncado)"
	if got := filetools.ReadFile(".", "over-cap.txt"); got != want {
		t.Errorf("ReadFile(over cap) returned %d bytes, want %d cut at the cap plus the marker", len(got), len(want))
	}
}

// TestReadFileTellsAbsenceApart pins the two read failures apart: an absent
// file comes back on READ_NOT_FOUND_NOTICE, carrying the path and the two
// things the turn must not do after it, while a file that exists and cannot be
// read (here a directory) keeps the blind error. The mutant that collapses the
// two cases back into "failed to read File content" fails the first arm.
func TestReadFileTellsAbsenceApart(t *testing.T) {
	genDir := filepath.Join("generated", "read")
	if err := os.MkdirAll(genDir, 0755); err != nil {
		t.Fatal(err)
	}

	originalWd, _ := os.Getwd()
	os.Chdir(genDir)
	defer os.Chdir(originalWd)

	if err := os.MkdirAll("a-directory", 0755); err != nil {
		t.Fatal(err)
	}

	absent := filetools.ReadFile(".", "no-such-file.conf")
	if want := filetools.READ_NOT_FOUND_NOTICE + "no-such-file.conf"; absent != want {
		t.Errorf("ReadFile(absent) = %q, want %q", absent, want)
	}
	for _, forbidden := range []string{"written", "created", "edited", "updated"} {
		if !strings.Contains(absent, forbidden) {
			t.Errorf("the absence notice does not name %q, so the turn is not told it may not claim it", forbidden)
		}
	}

	unreadable := filetools.ReadFile(".", "a-directory")
	if want := "Error: failed to read File content"; unreadable != want {
		t.Errorf("ReadFile(directory) = %q, want %q", unreadable, want)
	}
}

// TestReadToolDescriptionCarriesTheNotice keeps the description saying what the
// code does, as the edit and write notices already do.
func TestReadToolDescriptionCarriesTheNotice(t *testing.T) {
	if !strings.Contains(filetools.READ_TOOL_DESCRIPTION, filetools.READ_NOT_FOUND_NOTICE) {
		t.Error("READ_TOOL_DESCRIPTION does not show READ_NOT_FOUND_NOTICE to the model")
	}
}
