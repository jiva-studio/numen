package settings

import settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"

// CheckScale refuses a scale outside the range of its setting.
type CheckScale struct {
	bounds settingsdomain.Bounds
}

// NewCheckScale makes the use case over the range the scale setting declares.
func NewCheckScale(bounds settingsdomain.Bounds) *CheckScale {
	return &CheckScale{bounds: bounds}
}

// Execute returns an error wrapping settingsdomain.OutsideBoundsError for a scale outside the range.
// name is what the error calls the scale: a flag or a setting path.
func (u *CheckScale) Execute(name string, scale float64) error {
	return u.bounds.Check(name, scale)
}
