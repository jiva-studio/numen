package settings_test

import (
	"slices"
	"testing"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func TestListAppModelsPrefixesEveryModelAndWritePathWithTheAppOwner(t *testing.T) {
	list := settings.NewListAppModels(func() []settingsdomain.Model {
		return []settingsdomain.Model{{
			Path:   []string{"agent", "use"},
			Writes: []settingsdomain.Setting{{Path: []string{"agent", "use"}}, {Path: []string{"agent", "command"}}},
		}}
	})

	got := list.Execute()

	if len(got) != 1 || !slices.Equal(got[0].Path, []string{settingsdomain.AppOwner, "agent", "use"}) {
		t.Fatalf("model = %+v", got)
	}
	if !slices.Equal(got[0].Writes[1].Path, []string{settingsdomain.AppOwner, "agent", "command"}) {
		t.Errorf("write = %+v", got[0].Writes[1])
	}
}
