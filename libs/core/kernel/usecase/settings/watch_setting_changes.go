package settings

import (
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
)

// WatchSettingChanges follows the settings that change, whoever changes them: a write by a window,
// a plugin or the palette, and an edit of the file by hand.
type WatchSettingChanges struct {
	subscribe ChangeSubscriber
}

// NewWatchSettingChanges makes the use case over the subscription.
func NewWatchSettingChanges(subscribe ChangeSubscriber) *WatchSettingChanges {
	return &WatchSettingChanges{subscribe: subscribe}
}

// Execute starts listening. Each batch holds the settings that changed together, named by their
// declared path, once. Calling the returned function stops the delivery.
func (u *WatchSettingChanges) Execute() (<-chan []settingsdomain.Setting, func()) {
	return u.subscribe()
}
