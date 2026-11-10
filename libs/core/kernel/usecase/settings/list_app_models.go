package settings

import settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"

// ListAppModels lists the models of the settings the application owns. Each model and each write it
// makes is addressed under the application's owner, as the settings of a plugin are under the plugin.
type ListAppModels struct {
	models func() []settingsdomain.Model
}

// NewListAppModels returns the use case over models, which lists the models by their path inside the
// application's own settings.
func NewListAppModels(models func() []settingsdomain.Model) *ListAppModels {
	return &ListAppModels{models: models}
}

func (u *ListAppModels) Execute() []settingsdomain.Model {
	models := u.models()
	for place := range models {
		models[place].Path = append([]string{settingsdomain.AppOwner}, models[place].Path...)
		for index := range models[place].Writes {
			models[place].Writes[index].Path = append([]string{settingsdomain.AppOwner}, models[place].Writes[index].Path...)
		}
	}
	return models
}
