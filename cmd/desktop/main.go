//go:build !bindings

package main

import (
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"simple-one-api/internal/webui"
	"simple-one-api/pkg/appserver"
	"simple-one-api/pkg/initializer"
)

func main() {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("resolve user config directory: %v", err)
		return
	}
	configPath, err := resolveDesktopConfig(os.Args, userConfigDir)
	if err != nil {
		log.Printf("prepare desktop config: %v", err)
		return
	}
	if err := initializer.Setup(configPath); err != nil {
		log.Printf("initialize desktop: %v", err)
		return
	}
	defer initializer.Cleanup()

	err = wails.Run(&options.App{
		Title:                    "Simple One",
		Width:                    1280,
		Height:                   820,
		MinWidth:                 860,
		MinHeight:                620,
		BackgroundColour:         options.NewRGB(247, 247, 245),
		EnableDefaultContextMenu: false,
		SingleInstanceLock:       &options.SingleInstanceLock{UniqueId: "com.simple-one-api.desktop"},
		DragAndDrop:              &options.DragAndDrop{DisableWebViewDrop: true},
		AssetServer: &assetserver.Options{
			Assets:     webui.Assets(),
			Middleware: appserver.DesktopAssetMiddleware,
		},
	})
	if err != nil {
		log.Printf("run desktop: %v", err)
	}
}
