package derivedfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultconfig"
)

// openTabsFile is the file of the core folder that holds the open tabs.
const openTabsFile = "core/open-tabs.json"

// OpenTabsFiles keeps the open tabs of a vault in `.numen/core/open-tabs.json`.
type OpenTabsFiles struct {
	Options vaultconfig.Options
}

type openTabsRecord struct {
	Tabs  []tabRecord `json:"tabs"`
	Front string      `json:"front,omitempty"`
}

type tabRecord struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Path  string `json:"path,omitempty"`
	Title string `json:"title,omitempty"`
	State string `json:"state,omitempty"`
}

func (f OpenTabsFiles) ReadOpenTabs(ctx context.Context, v domain.Vault) (domain.OpenTabs, error) {
	store, err := OpenDerived(v.Path, f.Options, "core")
	if err != nil {
		return domain.OpenTabs{}, err
	}
	raw, err := store.Read(ctx, openTabsFile)
	if errors.Is(err, fs.ErrNotExist) {
		return domain.OpenTabs{}, nil
	}
	if err != nil {
		return domain.OpenTabs{}, err
	}
	var record openTabsRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return domain.OpenTabs{}, err
	}
	open := domain.OpenTabs{Tabs: make([]domain.Tab, 0, len(record.Tabs)), FrontID: record.Front}
	for _, one := range record.Tabs {
		open.Tabs = append(open.Tabs, domain.Tab(one))
	}
	return open, nil
}

func (f OpenTabsFiles) WriteOpenTabs(ctx context.Context, v domain.Vault, open domain.OpenTabs) error {
	store, err := OpenDerived(v.Path, f.Options, "core")
	if err != nil {
		return err
	}
	record := openTabsRecord{Tabs: make([]tabRecord, 0, len(open.Tabs)), Front: open.FrontID}
	for _, one := range open.Tabs {
		record.Tabs = append(record.Tabs, tabRecord(one))
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return store.Write(ctx, openTabsFile, raw)
}
