package wire

import (
	"context"
	"slices"
	"strings"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"
)

// ResolveAddresses answers where the addresses of a request and the wikilinks
// of its text land in one vault. An address that reaches nothing is left out of
// the resolved ones. Every window that resolves links answers with it.
func ResolveAddresses(
	ctx context.Context, resolve *note.ResolveLinks, v domain.Vault, r *v1.ResolveAddressesRequest,
) (*v1.ResolveAddressesResponse, error) {
	inText, addresses := getTextLinks(r.GetText())
	requested := append(slices.Clone(r.GetWritten()), addresses...)
	found, err := resolve.Execute(ctx, v, r.GetFrom(), requested)
	if err != nil {
		return nil, err
	}

	// In the order they were asked about, and an address asked about twice is
	// one answer.
	out := &v1.ResolveAddressesResponse{Found: inText}
	observed := make(map[string]bool, len(found))
	for _, content := range requested {
		one, reached := found[content]
		if !reached || observed[content] {
			continue
		}
		observed[content] = true
		vault, crossed := one.InVault(v.ID)
		out.Resolved = append(out.Resolved, &v1.ResolvedAddress{
			Written:     content,
			Path:        one.To,
			Vault:       string(vault),
			IsCrossed:   crossed,
			IsAmbiguous: one.IsAmbiguous,
		})
	}
	return out, nil
}

// getTextLinks is each wikilink of a text, in the order they are written,
// each raw form once, and the address of each, each address once. Code holds none.
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

// getWikilinkText is the text a wikilink reads as: what follows its `|`, or
// everything inside the brackets where it has none.
func getWikilinkText(inside string) string {
	if alias := strings.IndexByte(inside, '|'); alias >= 0 {
		return strings.TrimSpace(inside[alias+1:])
	}
	return strings.TrimSpace(inside)
}
