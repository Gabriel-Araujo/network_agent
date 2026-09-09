package config

import (
	"os"
	"path/filepath"
)

// fileLocation é o arquivo de configuração, gravado no home do usuário:
//
//	Windows:     C:\Users\{usuario}\.network_agent
//	Linux:       /home/{usuario}/.network_agent
//	macOS:       /Users/{usuario}/.network_agent
//
// os.UserHomeDir() resolve os três sistemas (USERPROFILE no Windows, HOME nos
// demais). Se o home não puder ser determinado, cai no diretório temporário
// para que a aplicação ainda consiga iniciar.
var fileLocation = defaultFileLocation()

func defaultFileLocation() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "network_agent")
	}
	return filepath.Join(home, ".network_agent")
}

// Path devolve o caminho do arquivo de configuração. Exportado para que cada
// front-end (CLI, UI) possa exibi-lo ao usuário.
func Path() string {
	return fileLocation
}

// AvailableProviders lista os tipos de provedor suportados. Cada front-end
// decide como apresentar a escolha (menu no terminal, select na GUI).
func AvailableProviders() []string {
	return []string{"OpenAI Compatible"}
}
