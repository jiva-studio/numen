package connection

import (
	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	connections "github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"google.golang.org/protobuf/types/known/structpb"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

func readFields(wire map[string]*structpb.Value) map[string]any {
	fields := make(map[string]any, len(wire))
	for key, value := range wire {
		fields[key] = value.AsInterface()
	}
	return fields
}

func writeFields(fields map[string]any) (map[string]*structpb.Value, error) {
	wire := make(map[string]*structpb.Value, len(fields))
	for key, value := range fields {
		converted, err := structpb.NewValue(value)
		if err != nil {
			return nil, err
		}
		wire[key] = converted
	}
	return wire, nil
}

func writeConnection(entry connections.Entry) (*v1.ProviderConnection, error) {
	fields, err := writeFields(entry.Fields)
	if err != nil {
		return nil, err
	}
	return &v1.ProviderConnection{
		Id:              entry.ID,
		TypeId:          entry.TypeID,
		Name:            entry.Name,
		Fields:          fields,
		SecretKeysSet:   entry.SecretKeysSet,
		IsTypeAvailable: entry.IsTypeAvailable,
		Revision:        int32(entry.Revision),
		IsComplete:      entry.IsComplete,
		Uses:            writeUses(entry.Uses),
	}, nil
}

func writeUses(uses []connection.Use) []*v1.ProviderConnectionUse {
	wire := make([]*v1.ProviderConnectionUse, 0, len(uses))
	for _, one := range uses {
		wire = append(wire, &v1.ProviderConnectionUse{PluginId: one.PluginID, PluginName: one.PluginName, Operations: one.Operations})
	}
	return wire
}

func writeType(one connection.Type) (*v1.ProviderConnectionType, error) {
	wire := &v1.ProviderConnectionType{
		TypeId:      one.GetID(),
		PluginId:    one.PluginID,
		PluginName:  one.PluginName,
		Title:       one.Declaration.Title,
		Description: one.Declaration.Description,
		Operations:  one.Declaration.Operations,
		IsAvailable: one.IsAvailable,
	}
	for _, field := range one.Declaration.Fields {
		converted, err := writeField(field)
		if err != nil {
			return nil, err
		}
		wire.Fields = append(wire.Fields, converted)
	}
	for _, field := range one.Declaration.RowFields {
		converted, err := writeField(field)
		if err != nil {
			return nil, err
		}
		wire.RowFields = append(wire.RowFields, converted)
	}
	return wire, nil
}

func writeField(field manifest.ConnectionField) (*v1.ManifestConnectionField, error) {
	wire := &v1.ManifestConnectionField{
		Key:         field.Key,
		Type:        field.Type,
		Title:       field.Title,
		Description: field.Description,
		FixedValue:  field.FixedValue,
		IsRequired:  field.IsRequired,
		IsAddress:   field.IsAddress,
		Min:         field.Min,
		Max:         field.Max,
		Step:        field.Step,
		OptionsFrom: field.OptionsFrom,
	}
	if field.Default != nil {
		value, err := structpb.NewValue(field.Default)
		if err != nil {
			return nil, err
		}
		wire.Default = value
	}
	if len(field.Options) > 0 {
		options, err := writeOptions(field)
		if err != nil {
			return nil, err
		}
		wire.Options = options
	}
	return wire, nil
}

func writeOptions(field manifest.ConnectionField) (*structpb.ListValue, error) {
	items := make([]any, 0, len(field.Options))
	for _, option := range field.Options {
		if option.Title == "" && option.Detail == "" {
			items = append(items, option.Value)
			continue
		}
		items = append(items, map[string]any{"value": option.Value, "title": option.Title, "detail": option.Detail})
	}
	return structpb.NewList(items)
}
