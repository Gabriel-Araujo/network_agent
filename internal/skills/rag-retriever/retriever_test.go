package ragretriever

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Gabriel-Araujo/network_agent/internal/agent/rag"
	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/pkg/util"
)

const inputExemplo = "meu ospf não converge entre o core (R1) e o edge (R2)"

// TestSaveResultsGravaNoHomeENaoNoCWD mede as duas direções: o espelho cai no
// diretório de dados do app, e o CWD do processo não recebe nada. Sem a
// segunda, "passa" não provaria que o CWD saiu do caminho — era exatamente o
// que o SafePath(".", …) fazia passar enquanto o arquivo nascia em
// cmd/ui/.agent/tmp/retrieval.
func TestSaveResultsGravaNoHomeENaoNoCWD(t *testing.T) {
	home := t.TempDir()
	t.Cleanup(config.SetDir(home))
	cwd := t.TempDir()
	t.Chdir(cwd)

	path, err := SaveResults(inputExemplo, []rag.Result{{Query: "OSPF MTU mismatch"}})
	if err != nil {
		t.Fatalf("SaveResults: %v", err)
	}

	name := util.Slugify(inputExemplo) + "-" + util.Hash8(inputExemplo) + ".json"
	want := filepath.Join(home, "tmp", "retrieval", name)
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("espelho não está no diretório de dados do app: %v", err)
	}

	entries, err := os.ReadDir(cwd)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("o CWD recebeu %d entrada(s), a primeira %q — want nenhuma", len(entries), entries[0].Name())
	}
}
