package presetwire

import (
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/flashcards"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/presets"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/wire/apierror"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/wire/filewire"
)

// NewDeckPresetResponse answers the read of the preset a deck is scheduled by. The title is looked
// up only for a preset that was read.
func NewDeckPresetResponse(
	found presets.Contents, err error, title func(path string) string,
) (*api.GetDeckPresetResponse, error) {
	if err != nil {
		return nil, apierror.NewFailure(err)
	}
	out := &api.GetDeckPresetResponse{
		Preset: nil, Error: api.ErrorCodeUnspecified, At: nil, Bounds: GetSettingsBounds(),
	}
	if reason, ok := apierror.ConvertOutcome(found.Outcome); ok {
		out.Error = reason
		return out, nil
	}
	out.Preset = ConvertPresetToWire(found, title(found.Path))
	// What the file was when this came out of it, for the client to present
	// when it writes the settings back.
	out.At = filewire.NewStat(found.FileStat)
	return out, nil
}

// NewReadPresetResponse answers the read of a preset. The title is looked up only for a preset that
// was read.
func NewReadPresetResponse(
	found presets.Contents, err error, title func(path string) string,
) (*api.ReadPresetResponse, error) {
	if err != nil {
		return nil, apierror.NewFailure(err)
	}
	out := &api.ReadPresetResponse{
		Preset: nil, Error: api.ErrorCodeUnspecified, At: nil, Bounds: GetSettingsBounds(),
	}
	if reason, ok := apierror.ConvertOutcome(found.Outcome); ok {
		out.Error = reason
		return out, nil
	}
	out.Preset = ConvertPresetToWire(found, title(found.Path))
	out.At = filewire.NewStat(found.FileStat)
	return out, nil
}

// NewVaultDeckPresetResponse answers the read of the preset a deck of a named vault is scheduled by.
func NewVaultDeckPresetResponse(
	found presets.Contents, err error, title func(path string) string,
) (*api.GetVaultDeckPresetResponse, error) {
	if err != nil {
		return nil, apierror.NewFailure(err)
	}
	out := &api.GetVaultDeckPresetResponse{Preset: nil, Error: api.ErrorCodeUnspecified}
	if reason, ok := apierror.ConvertOutcome(found.Outcome); ok {
		out.Error = reason
		return out, nil
	}
	out.Preset = ConvertPresetToWire(found, title(found.Path))
	return out, nil
}

// NewListResponse is every preset the vault holds, by path and by what it is called.
func NewListResponse(found []presets.Summary, err error) (*api.ListPresetsResponse, error) {
	if err != nil {
		return nil, apierror.NewFailure(err)
	}
	out := &api.ListPresetsResponse{Presets: make([]*api.PresetSummary, 0, len(found))}
	for _, one := range found {
		out.Presets = append(out.Presets, &api.PresetSummary{Path: one.Path, Title: one.Title})
	}
	return out, nil
}

// NewScheduleDeckResponse answers the end of the assignment of a deck to a preset.
func NewScheduleDeckResponse(pos domain.FileStat, err error) (*api.ScheduleDeckResponse, error) {
	out, err := filewire.ConvertWrite(pos, err, filewire.Reason{Err: presets.ErrNotAPreset, Code: api.ErrorCodeNotAPreset})
	if err != nil {
		return nil, err
	}
	return &api.ScheduleDeckResponse{Error: out.Error, At: out.At, IsIndexOutdated: out.IsIndexOutdated}, nil
}

// NewWritePresetResponse answers the end of the write of settings into a preset. A value outside
// what a preset may hold is the client's to correct.
func NewWritePresetResponse(pos domain.FileStat, err error) (*api.WritePresetResponse, error) {
	if errors.Is(err, presets.ErrOutOfBounds) {
		return nil, apierror.NewError(server.CodeInvalidArgument, err)
	}
	out, err := filewire.ConvertWrite(pos, err, filewire.Reason{Err: presets.ErrNotAPreset, Code: api.ErrorCodeNotAPreset})
	if err != nil {
		return nil, err
	}
	return &api.WritePresetResponse{Error: out.Error, At: out.At, IsIndexOutdated: out.IsIndexOutdated}, nil
}

// NewCurveResponse is the curve the settings amount to.
func NewCurveResponse(curve flashcards.Curve, err error) (*api.ComputeCurveResponse, error) {
	if err != nil {
		return nil, apierror.NewFailure(err)
	}
	return &api.ComputeCurveResponse{Curve: ConvertCurveToWire(curve)}, nil
}
