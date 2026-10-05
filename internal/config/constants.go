package config

import (
	"os"
	"path/filepath"
)

// configDirName é o diretório de dados do aplicativo no home do usuário. Ele
// guarda a configuração (config.json) e, futuramente, o histórico de conversas.
const configDirName = ".network_agent"

// configFileName é o arquivo de configuração dentro do diretório acima.
const configFileName = "config.json"

// fileLocation é o arquivo de configuração, gravado no home do usuário:
//
//	Windows:     C:\Users\{usuario}\.network_agent\config.json
//	Linux:       /home/{usuario}/.network_agent/config.json
//	macOS:       /Users/{usuario}/.network_agent/config.json
//
// os.UserHomeDir() resolve os três sistemas (USERPROFILE no Windows, HOME nos
// demais). Se o home não puder ser determinado, cai no diretório temporário
// para que a aplicação ainda consiga iniciar.
var fileLocation = defaultFileLocation()

// defaultFileLocation devolve o caminho do arquivo de configuração. O
// diretório que o contém é criado sob demanda: Save faz o MkdirAll na primeira
// gravação.
func defaultFileLocation() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "network_agent", configFileName)
	}
	return filepath.Join(home, configDirName, configFileName)
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
