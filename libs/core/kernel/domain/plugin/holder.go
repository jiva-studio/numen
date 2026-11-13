package plugin

import (
	"context"
	"strings"
)

// Holder is who holds a lease on a plugin process and why.
type Holder struct {
	Who string
	Why string
}

// Caller is where a call into the core came from: who made it and the call it came through.
type Caller struct {
	Who  string
	Call string
}

type callerKey struct{}

type purposeKey struct{}

// WithCaller returns a context whose leases are held in the name of the caller.
func WithCaller(ctx context.Context, caller Caller) context.Context {
	return context.WithValue(ctx, callerKey{}, caller)
}

// WithPurpose returns a context whose leases are held for the purpose, in place of the reason the
// adapter that takes the lease gives.
func WithPurpose(ctx context.Context, purpose string) context.Context {
	return context.WithValue(ctx, purposeKey{}, purpose)
}

// CoreHolder is a lease the core takes for itself.
func CoreHolder(why string) Holder {
	return Holder{Who: "core", Why: why}
}

// GetHolder returns the holder of a lease taken under ctx: the caller of ctx, the core where there is
// none, and the purpose of ctx where there is one, why otherwise.
func GetHolder(ctx context.Context, why string) Holder {
	holder := CoreHolder(why)
	if caller, ok := ctx.Value(callerKey{}).(Caller); ok && caller.Who != "" {
		holder.Who = caller.Who
		if caller.Call != "" {
			holder.Who += " via " + caller.Call
		}
	}
	if purpose, ok := ctx.Value(purposeKey{}).(string); ok && purpose != "" {
		holder.Why = purpose
	}
	return holder
}

// GetCallName returns the short name of the RPC a Connect path names: "Search" for
// "/numen.v1.HostSearchService/Search".
func GetCallName(path string) string {
	trimmed := strings.Trim(path, "/")
	if i := strings.LastIndex(trimmed, "/"); i >= 0 {
		return trimmed[i+1:]
	}
	return trimmed
}
