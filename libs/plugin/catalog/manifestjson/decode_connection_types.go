package manifestjson

import (
	"fmt"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func newConnectionTypes(wire []*v1.ManifestConnectionType) ([]manifest.ConnectionType, error) {
	var out []manifest.ConnectionType
	for _, one := range wire {
		fields, err := newConnectionFields(one.GetFields())
		if err != nil {
			return nil, fmt.Errorf("connection type %q: %w", one.GetId(), err)
		}
		rowFields, err := newConnectionFields(one.GetRowFields())
		if err != nil {
			return nil, fmt.Errorf("connection type %q: row %w", one.GetId(), err)
		}
		out = append(out, manifest.ConnectionType{
			ID:          one.GetId(),
			Title:       one.GetTitle(),
			Description: one.GetDescription(),
			Operations:  one.GetOperations(),
			Fields:      fields,
			RowFields:   rowFields,
			IsBuiltIn:   one.GetIsBuiltIn(),
		})
	}
	return out, nil
}

func newConnectionFields(wire []*v1.ManifestConnectionField) ([]manifest.ConnectionField, error) {
	var out []manifest.ConnectionField
	for _, one := range wire {
		options, err := newSettingOptions(one.GetOptions())
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", one.GetKey(), err)
		}
		out = append(out, manifest.ConnectionField{
			Key:         one.GetKey(),
			Type:        one.GetType(),
			Title:       one.GetTitle(),
			Description: one.GetDescription(),
			Default:     one.GetDefault().AsInterface(),
			FixedValue:  one.FixedValue,
			IsRequired:  one.GetIsRequired(),
			IsAddress:   one.GetIsAddress(),
			Options:     options,
			Min:         one.Min,
			Max:         one.Max,
			Step:        one.Step,
			OptionsFrom: one.GetOptionsFrom(),
		})
	}
	return out, nil
}
