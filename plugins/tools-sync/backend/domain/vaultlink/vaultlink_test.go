package vaultlink_test

import (
	"encoding/json"
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/vaultlink"
)

func TestLink_HasTheKeysOfLinkJSON(t *testing.T) {
	link := vaultlink.Link{
		RemoteVaultID: "0a0b0c", SealedKey: []byte{1, 2, 3}, CatalogName: "Test", ServerSeq: 7,
		FeedCursor: vaultlink.FeedCursor{FeedID: "feed-1", Seq: 9},
	}

	raw, err := json.Marshal(link)
	if err != nil {
		t.Fatal(err)
	}

	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"remoteVaultId", "sealedKey", "catalogName", "serverSeq", "feedCursor"} {
		if _, ok := keys[key]; !ok {
			t.Errorf("link.json has no key %q in %s", key, raw)
		}
	}
	if len(keys) != 5 {
		t.Errorf("link.json has %d keys, want 5: %s", len(keys), raw)
	}
	var cursor map[string]any
	if err := json.Unmarshal(keys["feedCursor"], &cursor); err != nil {
		t.Fatal(err)
	}
	if cursor["feedId"] != "feed-1" || cursor["seq"] != float64(9) {
		t.Errorf("feedCursor = %v, want feedId feed-1 and seq 9", cursor)
	}
}

func TestLink_ReadsBackWhatItWrote(t *testing.T) {
	link := vaultlink.Link{
		RemoteVaultID: "0a0b0c", SealedKey: []byte{1, 2, 3}, CatalogName: "Test", ServerSeq: 7,
		FeedCursor: vaultlink.FeedCursor{FeedID: "feed-1", Seq: 9},
	}
	raw, err := json.Marshal(link)
	if err != nil {
		t.Fatal(err)
	}

	var read vaultlink.Link
	if err := json.Unmarshal(raw, &read); err != nil {
		t.Fatal(err)
	}

	if read.RemoteVaultID != link.RemoteVaultID || string(read.SealedKey) != string(link.SealedKey) || read.CatalogName != "Test" ||
		read.ServerSeq != 7 || read.FeedCursor != link.FeedCursor {
		t.Errorf("link = %+v, want %+v", read, link)
	}
}
