package util

import (
	"testing"
)

// TestSlugify migrou de tests/internal/skills (intent-analyser) junto dos
// helpers — ver .agent/classifier/issues/04-desenho-da-skill-do-briefing-rico.md.
func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"meu ospf não converge entre o core", "meu-ospf-nao-converge-entre"},
		{"OSPF Não  Converge!!! 123", "ospf-nao-converge-123"},
		{"Gi0/1 em área 0", "gi0-1-em-area-0"},
		{"   ", ""},
		{"alô, mundo", "alo-mundo"},
	}
	for _, c := range cases {
		if got := Slugify(c.in); got != c.want {
			t.Errorf("slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHash8(t *testing.T) {
	h := Hash8("x")
	if len(h) != 8 {
		t.Errorf("hash8 len = %d, want 8", len(h))
	}
	if Hash8("x") == Hash8("y") {
		t.Errorf("hash8 colidiu para entradas diferentes")
	}
	// Determinístico: o mesmo input regrava o mesmo espelho.
	if Hash8("x") != h {
		t.Errorf("hash8 não é determinístico: %q != %q", Hash8("x"), h)
	}
}

func TestSafePath(t *testing.T) {
	tests := []struct {
		workingDir string
		filePath   string
		expected   string
		wantErr    bool
	}{
		{
			workingDir: ".",
			filePath:   "config/r1.cfg",
			expected:   "config/r1.cfg",
			wantErr:    false,
		},
		{
			workingDir: "/tmp",
			filePath:   "test.txt",
			expected:   "test.txt",
			wantErr:    false,
		},
		{
			workingDir: ".",
			filePath:   "../etc/passwd",
			expected:   "",
			wantErr:    true,
		},
		{
			workingDir: "/tmp",
			filePath:   "../etc/passwd",
			expected:   "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.workingDir+":"+tt.filePath, func(t *testing.T) {
			got, err := SafePath(tt.workingDir, tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("SafePath() got = %v, want %v", got, tt.expected)
			}
		})
	}
}
