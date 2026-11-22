//go:build !linux

package app

import "github.com/jiva-studio/numen/libs/core/kernel/container"

func applyWebviewEnvironment(container.Config) {}
