//go:build bindings

package main

import (
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

// Wails executes a bindings-tagged helper binary while generating JavaScript
// bindings. The desktop app exposes no native bindings, so keep that helper
// side-effect free and avoid creating a user configuration during builds.
func main() {
	_ = wails.Run(&options.App{})
}
