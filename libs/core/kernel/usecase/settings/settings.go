package settings

import settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"

// ChangeSubscriber starts listening to the settings that change. Each batch holds the settings that
// changed together, with the value each now holds, named by its declared path and delivered once
// to every subscriber. A path no owner declares is not delivered. The returned function stops the
// delivery and closes the channel.
type ChangeSubscriber func() (changes <-chan []settingsdomain.Setting, stop func())
