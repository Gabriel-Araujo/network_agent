package config

import (
	"os"
	"path/filepath"
)

var fileLocation = filepath.Join(os.TempDir(), "network_agent")

var availableProviders = []string{"OpenAI Compatible"}
