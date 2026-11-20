package types

import (
	"slices"
	"testing"
)

func TestUseWorkspaceOrdemDuplicataECorte(t *testing.T) {
	var c Config
	for _, d := range []string{`C:\a`, `C:\b`, `C:\c`, `C:\d`, `C:\e`} {
		c.UseWorkspace(d)
	}
	if want := []string{`C:\e`, `C:\d`, `C:\c`, `C:\b`, `C:\a`}; !slices.Equal(c.RecentWorkspaces, want) {
		t.Fatalf("ordem = %v, want %v", c.RecentWorkspaces, want)
	}

	// Reusar um recente, mesmo em outra caixa e com barra no fim, sobe sem duplicar.
	c.UseWorkspace(`c:\B\`)
	if want := []string{`c:\B\`, `C:\e`, `C:\d`, `C:\c`, `C:\a`}; !slices.Equal(c.RecentWorkspaces, want) {
		t.Fatalf("duplicata = %v, want %v", c.RecentWorkspaces, want)
	}

	// O sexto entra e o mais antigo sai.
	c.UseWorkspace(`C:\f`)
	if len(c.RecentWorkspaces) != MaxRecentWorkspaces || c.RecentWorkspaces[0] != `C:\f` ||
		slices.Contains(c.RecentWorkspaces, `C:\a`) || c.Workspace != `C:\f` {
		t.Fatalf("corte = %v (Workspace %q)", c.RecentWorkspaces, c.Workspace)
	}
}
