package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
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
		println("Error:", err.Error())
	}
}
