package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	filetools "github.com/Gabriel-Araujo/network_agent/internal/tools/file"
)

func TestWriteFile(t *testing.T) {
	// Define the generated directory path
	genDir := filepath.Join("generated", "write")
	err := os.MkdirAll(genDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Clean up generated files from previous runs so the new-file cases
	// really create their file instead of replacing it.
	files, _ := filepath.Glob(filepath.Join(genDir, "*"))
	for _, f := range files {
		os.RemoveAll(f)
	}

	originalWd, _ := os.Getwd()
	os.Chdir(genDir)
	defer os.Chdir(originalWd)

	wd, _ := os.Getwd()
	outsideAbs := filepath.Join(filepath.Dir(wd), "outside-abs.txt")
	os.Remove(outsideAbs)

	tests := []struct {
		name       string
		workingDir string
		filePath   string
		content    string
		want       string
	}{
		{
			name:       "Success write new file",
			workingDir: ".",
			filePath:   "new.txt",
			// As duas direções do ticket 22: os dois sucessos têm a mesma
			// forma e verbos diferentes. O sucesso de arquivo novo devolvia
			// só o conteúdo, e o modelo que não vê primeira linha nenhuma
			// inventa a pasta ("no diretório atual") e hedgeia a
			// substituição.
			content: "new content",
			want:    "Created new file: " + filepath.ToSlash(filepath.Join(wd, "new.txt")) + "\nnew content",
		},
		{
			name:       "Success write in new subdirectory",
			workingDir: ".",
			filePath:   "sub/dir/new.txt",
			content:    "sub content",
			want:       "Created new file: " + filepath.ToSlash(filepath.Join(wd, "sub/dir/new.txt")) + "\nsub content",
		},
		{
			name:       "Success replace existing file",
			workingDir: ".",
			filePath:   "existing.txt",
			content:    "replaced content",
			// SafePath devolve o absoluto contido, e é ele que o aviso de
			// substituição mostra: diz onde a tool realmente gravou.
			want: "Replaced existing file: " + filepath.ToSlash(filepath.Join(wd, "existing.txt")) + "\nreplaced content",
		},
		{
			name:       "Error path outside",
			workingDir: ".",
			filePath:   "../outside.txt",
			content:    "fail",
			want:       "Error Filepath outside the permitted working directory",
		},
		{
			name:       "Error absolute path outside",
			workingDir: ".",
			filePath:   outsideAbs,
			content:    "fail",
			want:       "Error Filepath outside the permitted working directory",
		},
	}

	// Set up the existing file the replace case overwrites
	os.WriteFile("existing.txt", []byte("exists"), 0644)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := filetools.WriteFile(tt.workingDir, tt.filePath, tt.content)
			if strings.HasPrefix(tt.want, "Error") {
				if got != tt.want {
					t.Errorf("WriteFile() = %q, want %q", got, tt.want)
				}
				if _, err := os.Stat(outsideAbs); err == nil {
					t.Errorf("WriteFile() created %s outside the working directory", outsideAbs)
				}
			} else {
				if got != tt.want {
					t.Errorf("WriteFile() = %q, want %q", got, tt.want)
				}
				// Verify file actually written
				data, err := os.ReadFile(filepath.Join(tt.workingDir, tt.filePath))
				if err != nil {
					t.Errorf("Failed to read back written file: %v", err)
				}
				if string(data) != tt.content {
					t.Errorf("File content = %q, want %q", string(data), tt.content)
				}
			}
		})
	}
}

// TestWriteFileHonraDiretorioDeTrabalhoAbsoluto fixa o contrato que este
// ticket existe para consertar: com um diretório de trabalho que NÃO é o CWD
// do processo, o arquivo cai no diretório de trabalho, não no CWD. Antes de o
// SafePath devolver o absoluto contido ele caía no CWD — pasta errada, sem
// erro. Todo o resto da suíte usa workingDirectory ".", onde o relativo e o
// absoluto coincidem e o defeito não aparece. É a pré-condição da pasta
// escolhida pelo usuário (.agent/ui/issues/05-desenho-do-workspace-de-escrita.md).
func TestWriteFileHonraDiretorioDeTrabalhoAbsoluto(t *testing.T) {
	workspace := t.TempDir()
	cwd := t.TempDir()
	t.Chdir(cwd)

	const content = "hostname r1"
	// A primeira linha é a única coisa que diz contra qual raiz o relativo
	// resolveu, e aqui as duas candidatas divergem: o workspace não é o CWD.
	want := "Created new file: " + filepath.ToSlash(filepath.Join(workspace, "r1.cfg")) + "\n" + content
	if got, _ := filetools.WriteFile(workspace, "r1.cfg", content); got != want {
		t.Fatalf("WriteFile() = %q, want %q", got, want)
	}

	data, err := os.ReadFile(filepath.Join(workspace, "r1.cfg"))
	if err != nil {
		t.Fatalf("não gravou dentro do diretório de trabalho: %v", err)
	}
	if string(data) != content {
		t.Errorf("conteúdo = %q, want %q", data, content)
	}
	if _, err := os.Stat(filepath.Join(cwd, "r1.cfg")); err == nil {
		t.Error("gravou no CWD do processo em vez do diretório de trabalho")
	}

	// A leitura resolve pela mesma raiz, não pelo CWD.
	if got, _ := filetools.ReadFile(workspace, "r1.cfg"); got != content {
		t.Errorf("ReadFile() = %q, want o arquivo do diretório de trabalho", got)
	}
}
