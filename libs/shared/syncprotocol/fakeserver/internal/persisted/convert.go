package persisted

import (
	"encoding/hex"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver/internal/sharestate"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
)

// DecodeID reads an id that the snapshot keeps as hex.
func DecodeID(encoded string) (string, error) {
	raw, err := hex.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("read an id of the snapshot: %w", err)
	}
	return string(raw), nil
}

// EncodeShare returns the JSON form of a share.
func EncodeShare(share *sharestate.State) (Share, error) {
	kept := Share{
		Account: share.Account, Vault: hex.EncodeToString([]byte(share.Vault)), Epoch: share.Epoch, IsClosed: share.IsClosed, Members: share.Members,
		Exits: map[string]uint64{},
	}
	for file, seq := range share.Exits {
		kept.Exits[hex.EncodeToString([]byte(file))] = seq
	}
	for _, entry := range share.Log {
		raw, err := proto.Marshal(entry)
		if err != nil {
			return Share{}, fmt.Errorf("keep a share log entry: %w", err)
		}
		kept.Log = append(kept.Log, raw)
	}
	return kept, nil
}

// DecodeShares reads the shares of a snapshot, by their ids.
func DecodeShares(in map[string]Share) (map[string]*sharestate.State, error) {
	out := map[string]*sharestate.State{}
	for encodedID, kept := range in {
		id, err := DecodeID(encodedID)
		if err != nil {
			return nil, err
		}
		vault, err := DecodeID(kept.Vault)
		if err != nil {
			return nil, err
		}
		share := sharestate.New(kept.Account, vault)
		share.Epoch, share.IsClosed, share.Members = kept.Epoch, kept.IsClosed, kept.Members
		for encodedFile, seq := range kept.Exits {
			file, err := DecodeID(encodedFile)
			if err != nil {
				return nil, err
			}
			share.Exits[file] = seq
		}
		for _, raw := range kept.Log {
			entry := &syncv1.ShareLogEntry{}
			if err := proto.Unmarshal(raw, entry); err != nil {
				return nil, fmt.Errorf("read a share log entry: %w", err)
			}
			share.Log = append(share.Log, entry)
		}
		out[id] = share
	}
	return out, nil
}

// DecodeChains reads the account chains of a snapshot.
func DecodeChains(in map[string][]byte) (map[string]identity.Chain, error) {
	out := map[string]identity.Chain{}
	for account, raw := range in {
		chain, err := identity.ParseChain(raw)
		if err != nil {
			return nil, fmt.Errorf("read a chain: %w", err)
		}
		out[account] = chain
	}
	return out, nil
}

// DecodeCatalogs reads the catalog entries of a snapshot.
func DecodeCatalogs(in map[string][][]byte) (map[string][]*syncv1.CatalogEntry, error) {
	out := map[string][]*syncv1.CatalogEntry{}
	for account, raws := range in {
		for _, raw := range raws {
			entry := &syncv1.CatalogEntry{}
			if err := proto.Unmarshal(raw, entry); err != nil {
				return nil, fmt.Errorf("read a catalog entry: %w", err)
			}
			out[account] = append(out[account], entry)
		}
	}
	return out, nil
}
