package plugin

import (
	"context"
	"slices"
	"strings"
	"unicode"
)

// The parts of Numen that hold leases on plugins, as a person reads them.
const (
	WhoNumen       = "Numen"
	WhoWindow      = "Window"
	WhoIndexing    = "Indexing"
	WhoEvents      = "Events"
	WhoDetection   = "Connection detection"
	WhoListeners   = "Listeners"
	WhoSearch      = "Search"
	WhoSettings    = "Settings"
	WhoConnections = "Connections"
	WhoPluginStart = "Plugin start"
)

// Holder is who holds a lease on a plugin process and why. Who names a part of Numen. PluginID and
// ConnectionID name a plugin or a connection that asked, by id; the list of processes shows their names
// in place of Who.
type Holder struct {
	Who          string
	PluginID     string
	ConnectionID string
	Why          string
}

// Caller is where a call into the core came from: a part of Numen, or a plugin, or the connection a
// plugin calls for.
type Caller struct {
	Who          string
	PluginID     string
	ConnectionID string
}

func (c Caller) isSet() bool {
	return c.Who != "" || c.PluginID != "" || c.ConnectionID != ""
}

type callerKey struct{}

type purposeKey struct{}

// WithCaller returns a context whose leases are held in the name of the caller.
func WithCaller(ctx context.Context, caller Caller) context.Context {
	return context.WithValue(ctx, callerKey{}, caller)
}

// WithBackground returns a context whose leases are held in the name of a part of Numen that works in
// the background.
func WithBackground(ctx context.Context, who string) context.Context {
	return WithCaller(ctx, Caller{Who: who})
}

// WithPurpose returns a context whose leases are held for the purpose, in place of the reason the
// adapter that takes the lease gives.
func WithPurpose(ctx context.Context, purpose string) context.Context {
	return context.WithValue(ctx, purposeKey{}, purpose)
}

// GetCaller returns the caller of ctx, and false where it has none.
func GetCaller(ctx context.Context) (Caller, bool) {
	caller, ok := ctx.Value(callerKey{}).(Caller)
	return caller, ok && caller.isSet()
}

// GetHolder returns the holder of a lease taken under ctx: the caller of ctx, Numen where there is
// none, and the purpose of ctx where there is one, why otherwise.
func GetHolder(ctx context.Context, why string) Holder {
	holder := Holder{Who: WhoNumen, Why: why}
	if caller, ok := GetCaller(ctx); ok {
		holder.Who, holder.PluginID, holder.ConnectionID = caller.Who, caller.PluginID, caller.ConnectionID
	}
	if purpose, ok := ctx.Value(purposeKey{}).(string); ok && purpose != "" {
		holder.Why = purpose
	}
	return holder
}

// GetCallTitle returns the method a Connect path or a plugin method path names, in plain words: "list
// models" for "/llm.ModelService/ListModels".
func GetCallTitle(path string) string {
	trimmed := strings.Trim(path, "/")
	if i := strings.LastIndex(trimmed, "/"); i >= 0 {
		trimmed = trimmed[i+1:]
	}
	var words []string
	start := 0
	runes := []rune(trimmed)
	for i := 1; i < len(runes); i++ {
		if unicode.IsUpper(runes[i]) && !unicode.IsUpper(runes[i-1]) || runes[i] == '_' || runes[i] == '-' {
			words = append(words, string(runes[start:i]))
			start = i
			if runes[i] == '_' || runes[i] == '-' {
				start = i + 1
			}
		}
	}
	words = append(words, string(runes[start:]))
	words = slices.DeleteFunc(words, func(word string) bool { return word == "" })
	return strings.ToLower(strings.Join(words, " "))
}

// WithAskingPlugin returns a context whose leases are held in the name of the plugin, where ctx names no
// caller or the window as a whole: a view of the plugin asked.
func WithAskingPlugin(ctx context.Context, pluginID string) context.Context {
	if caller, ok := GetCaller(ctx); ok && caller != (Caller{Who: WhoWindow}) {
		return ctx
	}
	return WithCaller(ctx, Caller{PluginID: pluginID})
}
