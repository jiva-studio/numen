package delivery

import (
	"context"
	"errors"
	"fmt"
	"strings"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// allowedList is one declaration of a plugin with the secrets allowed to it, in the order they are sent.
type allowedList struct {
	declaration manifest.SecretDeclaration
	secrets     []plugindomain.Secret
}

func listAllowedLists(snapshot Snapshot) ([]allowedList, error) {
	if len(snapshot.Declared) == 0 {
		return nil, nil
	}
	var list []plugindomain.Secret
	var listErr error
	if snapshot.source.List != nil {
		if list, listErr = snapshot.source.List(); listErr != nil {
			list = nil
			listErr = fmt.Errorf("read the secrets of plugin %s: %w", snapshot.PluginID, listErr)
		}
	}
	page := plugindomain.ListSecretsInPageOrder(list, snapshot.Requested)
	lists := make([]allowedList, 0, len(snapshot.Declared))
	for _, declaration := range snapshot.Declared {
		lists = append(lists, allowedList{
			declaration: declaration,
			secrets: plugindomain.ListAllowedSecrets(page, plugindomain.DeclarationKey{
				Plugin: snapshot.PluginID, Name: declaration.Name, Group: declaration.Group,
			}),
		})
	}
	return lists, listErr
}

func readEntries(ctx context.Context, snapshot Snapshot, lists []allowedList) ([]*v1.ConfigureSecrets, []error) {
	entries := make([]*v1.ConfigureSecrets, 0, len(lists))
	var failures []error
	for _, list := range lists {
		values, err := readSecretValues(ctx, snapshot.store, list.secrets)
		if err != nil {
			failures = append(failures, fmt.Errorf("read secret %s of plugin %s: %w", list.declaration.Name, snapshot.PluginID, err))
		}
		entries = append(entries, &v1.ConfigureSecrets{Name: list.declaration.Name, Group: list.declaration.Group, Values: values})
	}
	return entries, failures
}

func readSecretValues(ctx context.Context, store pluginport.KeyStore, secrets []plugindomain.Secret) ([]*v1.SecretValue, error) {
	values := make([]*v1.SecretValue, 0, len(secrets))
	for _, one := range secrets {
		if store == nil {
			return values, plugindomain.ErrNoKeyStore
		}
		value, err := store.GetKey(ctx, plugindomain.GetSecretEntryName(one.ID))
		switch {
		case err == nil:
			values = append(values, &v1.SecretValue{Id: one.ID, Name: one.Name, Group: one.Group, Value: value})
		case errors.Is(err, plugindomain.ErrNoKey):
		default:
			return values, err
		}
	}
	return values, nil
}

func getSignature(lists []allowedList) string {
	var signature strings.Builder
	for _, list := range lists {
		fmt.Fprintf(&signature, "%q/%q:", list.declaration.Name, list.declaration.Group)
		for _, one := range list.secrets {
			fmt.Fprintf(&signature, "%q/%q/%q,", one.ID, one.Name, one.Group)
		}
		signature.WriteString(";")
	}
	return signature.String()
}
