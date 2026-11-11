package wire

import (
	"context"
	"slices"
	"strings"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/markdown"
)

// ResolveAddresses resolves the addresses of a request and the wikilinks of its
// text within one vault. An address that resolves to nothing is omitted from the
// result. Every window that resolves links uses it.
func ResolveAddresses(
	ctx context.Context, resolve LinkResolver, vault domain.VaultID, r *v1.ResolveAddressesRequest,
) (*v1.ResolveAddressesResponse, error) {
	inText, addresses := getTextLinks(r.GetText())
	requested := append(slices.Clone(r.GetWritten()), addresses...)
	found, err := resolve.Execute(ctx, vault, r.GetFrom(), requested)
	if err != nil {
		return nil, err
	}

	// Results are in request order, and an address requested twice yields one
	// result.
	out := &v1.ResolveAddressesResponse{Found: inText}
	observed := make(map[string]bool, len(found))
	for _, content := range requested {
		one, reached := found[content]
		if !reached || observed[content] {
			continue
		}
		observed[content] = true
		resolvedVault, crossed := one.InVault(vault)
		out.Resolved = append(out.Resolved, &v1.ResolvedAddress{
			Written:     content,
			Path:        one.To,
			Vault:       string(resolvedVault),
			IsCrossed:   crossed,
			IsAmbiguous: one.IsAmbiguous,
		})
	}
	return out, nil
}

func getTextLinks(text string) ([]*v1.TextLink, []string) {
	var links []*v1.TextLink
	var addresses []string
	isListed := map[string]bool{}
	isAddressed := map[string]bool{}
	for _, link := range markdown.WikilinksInText(text) {
		raw := text[link.At:link.To]
		if isListed[raw] {
			continue
		}
		isListed[raw] = true
		address := link.Target.String()
		links = append(links, &v1.TextLink{
			Raw:     raw,
			Address: address,
			Text:    getWikilinkText(link.Inside),
		})
		if !isAddressed[address] {
			isAddressed[address] = true
			addresses = append(addresses, address)
		}
	}
	return links, addresses
}

func getWikilinkText(inside string) string {
	if _, after, ok := strings.Cut(inside, "|"); ok {
		return strings.TrimSpace(after)
	}
	return strings.TrimSpace(inside)
}

// LinkResolver finds where addresses written in one note of a vault land.
type LinkResolver interface {
	Execute(ctx context.Context, vaultID domain.VaultID, from string, targets []string) (map[string]domain.ResolvedLink, error)
}
