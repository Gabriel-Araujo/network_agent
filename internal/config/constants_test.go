package config

import (
	"path/filepath"
	"testing"
)

// TestDefaultFileLocation fixa o novo layout: a configuração é um ARQUIVO
// dentro do diretório de dados do aplicativo (~/.network_agent/config.json),
// não mais o próprio ~/.network_agent. Nenhum arquivo é tocado aqui — só a
// forma do caminho.
func TestDefaultFileLocation(t *testing.T) {
	loc := defaultFileLocation()

	if got := filepath.Base(loc); got != configFileName {
		t.Errorf("base de defaultFileLocation() = %q, want %q", got, configFileName)
	}

	// O diretório é o de dados do app no home; quando o home não resolve, o
	// fallback temporário mantém a mesma forma (network_agent/config.json).
	dir := filepath.Base(filepath.Dir(loc))
	if dir != configDirName && dir != "network_agent" {
		t.Errorf("diretório de defaultFileLocation() = %q, want %q ou %q", dir, configDirName, "network_agent")
	}
}

// TestPathIsConfigFileInsideConfigDir garante que Path() continua devolvendo o
// caminho do ARQUIVO de configuração — os front-ends exibem esse caminho em
// mensagens de erro — agora dentro do diretório de dados.
func TestPathIsConfigFileInsideConfigDir(t *testing.T) {
	got := Path()

	if got != fileLocation {
		t.Errorf("Path() = %q, want %q", got, fileLocation)
	}
	if base := filepath.Base(got); base != configFileName {
		t.Errorf("Path() = %q, want o arquivo %q no fim", got, configFileName)
	}
	if dir := filepath.Base(filepath.Dir(got)); dir != configDirName && dir != "network_agent" {
		t.Errorf("Path() = %q, want o diretório %q ou %q antes do arquivo", got, configDirName, "network_agent")
	}
}
