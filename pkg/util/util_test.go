package util

import (
	"testing"
)

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
