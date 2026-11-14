package pluginhealth

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// Clear takes a health item a plugin reported about itself out of the list.
type Clear struct {
	items *Items
}

// NewClear returns the use case over the list.
func NewClear(items *Items) *Clear {
	return &Clear{items: items}
}

// Execute removes the item of the plugin under the key. A key that has no item is already cleared. It
// returns plugin.ErrInvalidHealth for a key of the core and a key that is not a key.
func (u *Clear) Execute(pluginID, key string) error {
	if plugin.IsCoreHealthKey(key) {
		return fmt.Errorf("%w: the key %q is the core's", plugin.ErrInvalidHealth, key)
	}
	if err := plugin.CheckHealthKey(key); err != nil {
		return err
	}
	u.items.Remove(pluginID, key)
	return nil
}
