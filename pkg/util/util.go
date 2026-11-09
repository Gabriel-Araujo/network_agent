package util

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Gabriel-Araujo/network_agent/pkg/exception"
)

// SafePath returns filePath relative to workingDirectory, or InvalidFilePath
// when it resolves outside it. A path that names its own root or volume
// (C:\x, D:x, \\host\share\x; /x on Unix) resolves on its own and must land
// inside; any other path is joined to workingDirectory.
func SafePath(workingDirectory string, filePath string) (string, error) {
	absolutePath, err := filepath.Abs(workingDirectory)
	if err != nil {
		return "", exception.InvalidFilePath
	}

	targetPath := filepath.Join(absolutePath, filePath)
	if filepath.IsAbs(filePath) || filepath.VolumeName(filePath) != "" {
		targetPath, err = filepath.Abs(filePath)
		if err != nil {
			return "", exception.InvalidFilePath
		}
	}

	rel, err := filepath.Rel(absolutePath, targetPath)

	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return "", exception.InvalidFilePath
	}

	return rel, nil
}

func FormatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d bytes", size)
	}

	kb := float64(size) / 1024
	if kb < 1024 {
		s := fmt.Sprintf("%.1f", kb)
		s = strings.TrimSuffix(s, ".0")
		return s + "KB"
	}

	mb := kb / 1024
	if mb < 1024 {
		s := fmt.Sprintf("%.1f", mb)
		s = strings.TrimSuffix(s, ".0")
		return s + "MB"
	}

	gb := mb / 1024
	s := fmt.Sprintf("%.1f", gb)
	s = strings.TrimSuffix(s, ".0")
	return s + "GB"
}

// Slugify converte um input em um slug minúsculo, sem acentos, limitado a
// no máximo 5 palavras (separadas por hífen), contendo apenas [a-z0-9-].
func Slugify(input string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(input) {
		switch r {
		case 'á', 'à', 'â', 'ä', 'ã':
			r = 'a'
		case 'é', 'è', 'ê', 'ë':
			r = 'e'
		case 'í', 'ì', 'î', 'ï':
			r = 'i'
		case 'ó', 'ò', 'ô', 'ö', 'õ':
			r = 'o'
		case 'ú', 'ù', 'û', 'ü':
			r = 'u'
		case 'ç':
			r = 'c'
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}

	parts := strings.FieldsFunc(b.String(), func(r rune) bool { return r == '-' })
	if len(parts) > 5 {
		parts = parts[:5]
	}
	return strings.Join(parts, "-")
}

// Hash8 retorna os primeiros 8 caracteres hexadecimais do SHA1 do input.
// É determinístico de propósito: o mesmo input regrava o mesmo espelho
// <slug>-<hash8>.json (sobrescreve, sem histórico de execuções) — molde
// fixado em .agent/classifier/issues/04-desenho-da-skill-do-briefing-rico.md.
func Hash8(input string) string {
	h := sha1.Sum([]byte(input))
	return hex.EncodeToString(h[:])[:8]
}
