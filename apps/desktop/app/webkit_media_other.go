//go:build !(linux && cgo && !gtk3 && !android && !server && !purego)

package app

import "github.com/wailsapp/wails/v3/pkg/application"

func enableMediaCapture(*application.App) {}
