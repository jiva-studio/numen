// Package platform tells which operating system the plugin runs on.
package platform

import "runtime"

// IsPhone reports whether the plugin runs on a phone.
func IsPhone() bool { return IsPhoneSystem(runtime.GOOS) }

// IsPhoneSystem reports whether the operating system named goos is one of a phone.
func IsPhoneSystem(goos string) bool { return goos == "android" || goos == "ios" }
