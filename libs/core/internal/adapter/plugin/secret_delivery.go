package plugin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/core/usecase/secret"
)

// secretDelivery adds the secrets allowed to a plugin to its Configure call.
type secretDelivery struct {
	reportError    func(err error)
	keyStore       *keyStoreHolder
	mu             sync.RWMutex
	source         pluginport.SecretSource
	noKeyStoreOnce sync.Once
}

func newSecretDelivery(keyStore *keyStoreHolder, reportError func(err error)) *secretDelivery {
	return &secretDelivery{keyStore: keyStore, reportError: reportError}
}

func (d *secretDelivery) setSource(source pluginport.SecretSource) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.source = source
}

func (d *secretDelivery) getSource() pluginport.SecretSource {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.source
}

func (d *secretDelivery) newExtra(
	pluginID string, declared []manifest.SecretDeclaration, requested []string,
) func() configureExtra {
	store := d.keyStore.get()
	source := d.getSource()
	return func() configureExtra {
		lists, listErr := listAllowed(pluginID, declared, requested, source)
		return configureExtra{
			signature:           getSignature(lists),
			isComplete:          listErr == nil,
			isConfigureOptional: len(declared) == 0,
			read: func(ctx context.Context) ([]*v1.ConfigureSecrets, error) {
				entries, failures := readEntries(ctx, pluginID, store, lists)
				return entries, d.getReadError(listErr, failures)
			},
		}
	}
}

// allowedList is one declaration of a plugin with the secrets allowed to it, in the order they are sent.
type allowedList struct {
	declaration manifest.SecretDeclaration
	secrets     []pluginport.Secret
}

func listAllowed(
	pluginID string, declared []manifest.SecretDeclaration, requested []string, source pluginport.SecretSource,
) ([]allowedList, error) {
	if len(declared) == 0 {
		return nil, nil
	}
	var list []pluginport.Secret
	var listErr error
	if source.List != nil {
		if list, listErr = source.List(); listErr != nil {
			list = nil
			listErr = fmt.Errorf("read the secrets of plugin %s: %w", pluginID, listErr)
		}
	}
	page := secret.ListInPageOrder(list, requested)
	lists := make([]allowedList, 0, len(declared))
	for _, declaration := range declared {
		lists = append(lists, allowedList{
			declaration: declaration,
			secrets: secret.ListAllowed(page, pluginport.DeclarationKey{
				Plugin: pluginID, Name: declaration.Name, Group: declaration.Group,
			}),
		})
	}
	return lists, listErr
}

func readEntries(
	ctx context.Context, pluginID string, store pluginport.KeyStore, lists []allowedList,
) ([]*v1.ConfigureSecrets, []error) {
	entries := make([]*v1.ConfigureSecrets, 0, len(lists))
	var failures []error
	for _, list := range lists {
		values, err := readSecretValues(ctx, store, list.secrets)
		if err != nil {
			failures = append(failures, fmt.Errorf("read secret %s of plugin %s: %w", list.declaration.Name, pluginID, err))
		}
		entries = append(entries, &v1.ConfigureSecrets{Name: list.declaration.Name, Group: list.declaration.Group, Values: values})
	}
	return entries, failures
}

func readSecretValues(ctx context.Context, store pluginport.KeyStore, secrets []pluginport.Secret) ([]*v1.SecretValue, error) {
	values := make([]*v1.SecretValue, 0, len(secrets))
	for _, one := range secrets {
		if store == nil {
			return values, pluginport.ErrNoKeyStore
		}
		value, err := store.GetKey(ctx, secret.GetEntryName(one.ID))
		switch {
		case err == nil:
			values = append(values, &v1.SecretValue{Id: one.ID, Name: one.Name, Group: one.Group, Value: value})
		case errors.Is(err, pluginport.ErrNoKey):
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

func (d *secretDelivery) getReadError(listErr error, failures []error) error {
	problems := []error{listErr}
	for _, failure := range failures {
		if errors.Is(failure, pluginport.ErrNoKeyStore) {
			d.noKeyStoreOnce.Do(func() { d.reportError(failure) })
			continue
		}
		problems = append(problems, failure)
	}
	return errors.Join(problems...)
}
