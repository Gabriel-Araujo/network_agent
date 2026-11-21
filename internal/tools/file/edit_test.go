package filetools

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestEditFileNomeiaAsLinhasRemovidas fixa a regra 1 da decisão do ticket 08
// do mapa frr-file-builder: uma edição com sucesso devolve, ao modelo e ao app,
// as linhas do arquivo que saíram — e só elas. Os dois últimos casos medem a
// comparação por linha inteira: um old_text que pega meia linha nomeia a linha
// como estava, e um que come a quebra junta duas linhas, que somem as duas.
func TestEditFileNomeiaAsLinhasRemovidas(t *testing.T) {
	const conf = "router bgp 65001\n neighbor 203.0.113.2 remote-as 65010\n neighbor 203.0.113.2 description AS65010\n ! WARNING: no password on 203.0.113.2\n!\n"
	cases := []struct {
		nome, old, new string
		want           []string
	}{
		{"substituição de uma linha (o caso do 12B)",
			" neighbor 203.0.113.2 description AS65010", " neighbor 203.0.113.6 remote-as 65020",
			[]string{"neighbor 203.0.113.2 description AS65010"}},
		{"acréscimo que preserva a âncora",
			" neighbor 203.0.113.2 description AS65010\n", " neighbor 203.0.113.2 description AS65010\n neighbor 203.0.113.6 remote-as 65020\n",
			nil},
		{"só indentação, linha vazia e ! sozinho",
			" neighbor 203.0.113.2 description AS65010\n ! WARNING: no password on 203.0.113.2\n!\n", "  neighbor 203.0.113.2 description AS65010\n\n ! WARNING: no password on 203.0.113.2\n",
			nil},
		{"reordenar não é remover",
			" neighbor 203.0.113.2 remote-as 65010\n neighbor 203.0.113.2 description AS65010\n", " neighbor 203.0.113.2 description AS65010\n neighbor 203.0.113.2 remote-as 65010\n",
			nil},
		{"! WARNING: removido",
			" ! WARNING: no password on 203.0.113.2\n", "",
			[]string{"! WARNING: no password on 203.0.113.2"}},
		{"trecho no meio da linha nomeia a linha inteira",
			"AS65010", "AS65011",
			[]string{"neighbor 203.0.113.2 description AS65010"}},
		{"old_text que come a quebra junta duas linhas",
			"65001\n", "65001 ",
			[]string{"router bgp 65001", "neighbor 203.0.113.2 remote-as 65010"}},
	}
	for _, c := range cases {
		t.Run(c.nome, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "frr.conf"), []byte(conf), 0o644); err != nil {
				t.Fatal(err)
			}

			out, removed, err := EditFile(dir, "frr.conf", c.old, c.new)
			if err != nil {
				t.Fatalf("EditFile: %v (%s)", err, out)
			}
			if !slices.Equal(removed, c.want) {
				t.Errorf("removidas = %q, esperava %q", removed, c.want)
			}
			want := "Success: File edited"
			if c.want != nil {
				want += ". Removed lines: " + strings.Join(c.want, " | ")
			}
			if out != want {
				t.Errorf("resultado = %q, esperava %q", out, want)
			}
			got, _ := os.ReadFile(filepath.Join(dir, "frr.conf"))
			if string(got) != strings.Replace(conf, c.old, c.new, 1) {
				t.Errorf("frr.conf = %q", got)
			}
		})
	}
}
