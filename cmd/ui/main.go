package main

import (
	"embed"
	"path/filepath"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

var log = logger.Named("UI")

func main() {
	// O log da UI mora ao lado do config.json e do histórico, em
	// ~/.network_agent/logs/ui/. Um caminho relativo seria resolvido contra a
	// raiz do módulo (internal/logger/file.go:88-96), e aí o arquivo da
	// execução só existiria na máquina onde o binário foi compilado.
	cfg := logger.ConfigFromEnv()
	if cfg.File == "" {
		cfg.File = filepath.Join(filepath.Dir(config.Path()), "logs")
	}
	// Binary fixo: sob `wails dev` o os.Args[0] é o binário temporário da
	// geração de bindings, e cada nome abriria uma pasta própria. Fixo, as três
	// formas de rodar a UI (dev, binário construído, go run) gravam na mesma.
	cfg.Binary = "ui"
	// A gaveta de Logs é um segundo destino do sink de console, ligado aqui:
	// antes do Init, para o anel receber desde a primeira linha do run — as
	// quatro que nascem no NewApp(), que é onde as falhas de configuração e de
	// conexão com o provedor aparecem.
	cfg.ConsoleTee = logs
	logger.Init(cfg)

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "net_agent",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// Frameless: remove a barra de título branca do Windows. A barra é
		// redesenhada no frontend (topbar), com a cor do tema e os controles
		// de minimizar/maximizar/fechar próprios.
		Frameless: true,
		// Mesma cor de fundo do tema (--bg-base: #0a0f1c), para não haver
		// flash branco entre a criação da janela e o carregamento do webview.
		BackgroundColour: &options.RGBA{R: 0x0a, G: 0x0f, B: 0x1c, A: 1},
		OnStartup:        app.startup,
		Windows: &windows.Options{
			// Mantém o arrasto por área definida via CSS (--wails-draggable)
			// e o menu de sistema (Alt+Space, snap com Win+setas).
			DisableWindowIcon:    true,
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Error("o wails encerrou com erro", "err", err)
	}
}
