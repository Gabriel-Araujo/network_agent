package main

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/Gabriel-Araujo/network_agent/internal/config"
	"github.com/Gabriel-Araujo/network_agent/internal/logger"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
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

	wails := application.New(application.Options{
		Name:     "net_agent",
		Services: []application.Service{application.NewService(app)},
		Assets:   application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Windows: application.WindowsOptions{
			AdditionalBrowserArgs: debugArgs(),
			// Um perfil do WebView2 só, para o wails3 dev, o go run e o binário
			// lerem o mesmo localStorage (o tema). Sem isto, cada nome de exe
			// ganha uma pasta própria em %AppData%.
			WebviewUserDataPath: filepath.Join(filepath.Dir(config.Path()), "webview2"),
		},
	})

	window := wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "net_agent",
		Width:  1024,
		Height: 768,
		// Frameless: remove a barra de título branca do Windows. A barra é
		// redesenhada no frontend (topbar), com a cor do tema e os controles
		// de minimizar/maximizar/fechar próprios. O arrasto é por CSS
		// (--wails-draggable), e o menu de sistema (Alt+Space, snap com
		// Win+setas) continua.
		Frameless: true,
		// Nasce escondida e só aparece com a página carregada, já com o fundo
		// do tema pintado pelo script inline do index.html: sem flash.
		Hidden: true,
	})
	window.OnWindowEvent(events.Windows.WebViewNavigationCompleted, func(*application.WindowEvent) {
		window.Show()
	})

	if err := wails.Run(); err != nil {
		log.Error("o wails encerrou com erro", "err", err)
	}
}

// debugArgs abre a porta de CDP do WebView2 quando NETWORK_AGENT_CDP_PORT está
// definida. O Wails zera o WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS, então a
// variável do próprio WebView2 não tem efeito: é por aqui que a verificação por
// script chega à janela do app.
func debugArgs() []string {
	port := os.Getenv("NETWORK_AGENT_CDP_PORT")
	if port == "" {
		return nil
	}
	return []string{"--remote-debugging-port=" + port}
}
