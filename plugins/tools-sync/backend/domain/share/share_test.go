package share_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/identity"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/share"
)

func newKey(b byte) [32]byte {
	var key [32]byte
	for i := range key {
		key[i] = b
	}
	return key
}

func newMeta() note.Meta {
	return note.Meta{Path: "Dinner/Soup.md", Size: 12, MTime: 1_795_000_000, Hash: "abc", Key: bytes.Repeat([]byte{9}, 32), Kind: note.KindMarkdown}
}

func newChain(tb testing.TB) identity.Chain {
	tb.Helper()
	words, err := identity.NewRecoveryWords(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	root, err := words.DeriveRecoveryKey()
	if err != nil {
		tb.Fatal(err)
	}
	keys, err := identity.NewAccountKeys(rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	chain, err := identity.NewChain(root, keys, time.Date(2026, time.November, 23, 12, 0, 0, 0, time.UTC))
	if err != nil {
		tb.Fatal(err)
	}
	return chain
}

func TestCheckNesting_AcceptsAFolderApartFromTheSharedOnes(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		folder string
		shared []string
	}{
		"no share":                       {folder: "Food", shared: nil},
		"the vault root, nothing shared": {folder: "", shared: nil},
		"sibling":                        {folder: "Plans", shared: []string{"Food"}},
		"sibling with a common prefix":   {folder: "Foodie", shared: []string{"Food"}},
		"sibling the prefix is shared":   {folder: "Food", shared: []string{"Foodie"}},
		"nested sibling":                 {folder: "Life/Plans", shared: []string{"Life/Food"}},
	}
	for name, c := range cases {
		if err := share.CheckNesting(c.folder, c.shared); err != nil {
			t.Errorf("%s: error = %v, want none", name, err)
		}
	}
}

func TestCheckNesting_RejectsWholeVaultAndFolderNestedWithShare(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		folder string
		shared []string
		want   error
	}{
		"the vault root while a folder is shared": {folder: "", shared: []string{"Food"}, want: share.ErrHoldsShare},
		"the vault root as dot while one is":      {folder: ".", shared: []string{"Life/Food"}, want: share.ErrHoldsShare},
		"the vault root while the root is":        {folder: "", shared: []string{""}, want: share.ErrInsideShare},
		"a folder while the root is shared":       {folder: "Food", shared: []string{""}, want: share.ErrInsideShare},
		"the same folder":                         {folder: "Food", shared: []string{"Food"}, want: share.ErrInsideShare},
		"inside":                                  {folder: "Food/Dinner", shared: []string{"Food"}, want: share.ErrInsideShare},
		"deep inside":                             {folder: "Food/Dinner/Soup", shared: []string{"Food"}, want: share.ErrInsideShare},
		"inside with other case":                  {folder: "food/Dinner", shared: []string{"Food"}, want: share.ErrInsideShare},
		"holds":                                   {folder: "Life", shared: []string{"Life/Food"}, want: share.ErrHoldsShare},
		"holds with other case":                   {folder: "LIFE", shared: []string{"Life/Food"}, want: share.ErrHoldsShare},
		"inside the second":                       {folder: "Plans/Trip", shared: []string{"Food", "Plans"}, want: share.ErrInsideShare},
	}
	for name, c := range cases {
		if err := share.CheckNesting(c.folder, c.shared); !errors.Is(err, c.want) {
			t.Errorf("%s: error = %v, want %v", name, err, c.want)
		}
	}
}

func TestGetRelativePath_TrimsFolderPrefix(t *testing.T) {
	t.Parallel()
	for path, want := range map[string]string{"Food/Soup.md": "Soup.md", "Food/Dinner/Soup.md": "Dinner/Soup.md"} {
		got, ok := share.GetRelativePath("Food", path)
		if !ok || got != want {
			t.Errorf("%s: got %q (%v), want %q", path, got, ok, want)
		}
	}
}

func TestGetRelativePath_RefusesAFileOutsideTheFolder(t *testing.T) {
	t.Parallel()
	for _, path := range []string{"Soup.md", "Foodie/Soup.md", "Other/Food/Soup.md", "Food", "Food.md"} {
		if got, ok := share.GetRelativePath("Food", path); ok {
			t.Errorf("%s: got %q, want it outside the folder", path, got)
		}
	}
}

func TestGetShareName_IsTheLastSegmentOfTheFolder(t *testing.T) {
	t.Parallel()
	for folder, want := range map[string]string{"Food": "Food", "Life/Food": "Food", "Life/Food/Dinner": "Dinner"} {
		if got := share.GetShareName(folder); got != want {
			t.Errorf("%s: name = %q, want %q", folder, got, want)
		}
	}
}

func TestSealMeta_OpensWithTheKeyTheShareTheFileAndTheEpochOnly(t *testing.T) {
	t.Parallel()
	key, shareID, fileID := newKey(1), bytes.Repeat([]byte{2}, 16), bytes.Repeat([]byte{3}, 16)
	sealed, err := share.SealMeta(rand.Reader, key, shareID, fileID, 2, newMeta())
	if err != nil {
		t.Fatal(err)
	}

	opened, err := share.OpenMeta[note.Meta](key, shareID, fileID, 2, sealed)
	if err != nil || opened.Path != "Dinner/Soup.md" || !bytes.Equal(opened.Key, newMeta().Key) {
		t.Fatalf("opened = %+v, error = %v, want the meta that was sealed", opened, err)
	}
	for name, call := range map[string]func() error{
		"another key": func() error { _, err := share.OpenMeta[note.Meta](newKey(9), shareID, fileID, 2, sealed); return err },
		"another share": func() error {
			_, err := share.OpenMeta[note.Meta](key, bytes.Repeat([]byte{8}, 16), fileID, 2, sealed)
			return err
		},
		"another file": func() error {
			_, err := share.OpenMeta[note.Meta](key, shareID, bytes.Repeat([]byte{8}, 16), 2, sealed)
			return err
		},
		"another epoch": func() error { _, err := share.OpenMeta[note.Meta](key, shareID, fileID, 3, sealed); return err },
		"no bytes":      func() error { _, err := share.OpenMeta[note.Meta](key, shareID, fileID, 2, nil); return err },
	} {
		if err := call(); !errors.Is(err, share.ErrCannotOpen) {
			t.Errorf("%s: error = %v, want ErrCannotOpen", name, err)
		}
	}
}

func TestSealMeta_ShowsNeitherThePathNorTheKeyInTheSealedBytes(t *testing.T) {
	t.Parallel()
	meta := newMeta()
	sealed, err := share.SealMeta(rand.Reader, newKey(1), bytes.Repeat([]byte{2}, 16), bytes.Repeat([]byte{3}, 16), 0, meta)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("Soup")) || bytes.Contains(sealed, meta.Key) {
		t.Error("the sealed meta holds the path or the revision key in the clear")
	}
}

func TestSealMeta_IsNotOpenedByTheVaultKeyFunctionOfNotes(t *testing.T) {
	t.Parallel()
	key, vaultID, fileID := newKey(1), bytes.Repeat([]byte{2}, 16), bytes.Repeat([]byte{3}, 16)
	sealed, err := share.SealMeta(rand.Reader, key, vaultID, fileID, 0, newMeta())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := note.OpenMeta(key, vaultID, fileID, sealed); err == nil {
		t.Error("a share meta opened as a vault meta under the same key")
	}
}

func TestSealBody_OpensWithTheKeyTheShareAndTheEpochOnly(t *testing.T) {
	t.Parallel()
	key, shareID := newKey(1), bytes.Repeat([]byte{2}, 16)
	sealed, err := share.SealBody(rand.Reader, key, shareID, 1, share.Body{Name: "Food"})
	if err != nil {
		t.Fatal(err)
	}

	if body, err := share.OpenBody(key, shareID, 1, sealed); err != nil || body.Name != "Food" {
		t.Errorf("body = %+v, error = %v, want the name Food", body, err)
	}
	if _, err := share.OpenBody(key, shareID, 2, sealed); !errors.Is(err, share.ErrCannotOpen) {
		t.Errorf("another epoch: error = %v, want ErrCannotOpen", err)
	}
	if _, err := share.OpenBody(newKey(7), shareID, 1, sealed); !errors.Is(err, share.ErrCannotOpen) {
		t.Errorf("another key: error = %v, want ErrCannotOpen", err)
	}
}

func TestSealPreviousKey_SealsOlderKeyForNewerKeyHolder(t *testing.T) {
	t.Parallel()
	older, newer, shareID := newKey(1), newKey(2), bytes.Repeat([]byte{3}, 16)
	sealed, err := share.SealPreviousKey(rand.Reader, newer, older, shareID, 3)
	if err != nil {
		t.Fatal(err)
	}

	got, err := share.OpenPreviousKey(newer, shareID, 3, sealed)
	if err != nil || got != older {
		t.Fatalf("key = %x, error = %v, want the older key", got, err)
	}
	if _, err := share.OpenPreviousKey(older, shareID, 3, sealed); !errors.Is(err, share.ErrCannotOpen) {
		t.Errorf("the older key opened its own seal: error = %v", err)
	}
	if _, err := share.OpenPreviousKey(newer, shareID, 4, sealed); !errors.Is(err, share.ErrCannotOpen) {
		t.Errorf("another epoch: error = %v, want ErrCannotOpen", err)
	}
}

func TestGetSealContext_FollowsTheFrozenLayoutAndSeparatesShareReaderAndEpoch(t *testing.T) {
	t.Parallel()
	shareID, reader := bytes.Repeat([]byte{1}, 16), bytes.Repeat([]byte{2}, 32)

	got := share.GetSealContext(shareID, reader, 5)

	want := append(append(append([]byte("numen-sync/share-seal/v1"), shareID...), reader...), 0, 0, 0, 5)
	if !bytes.Equal(got, want) {
		t.Errorf("context = %x, want %x", got, want)
	}
	for name, other := range map[string][]byte{
		"share":  share.GetSealContext(bytes.Repeat([]byte{9}, 16), reader, 5),
		"reader": share.GetSealContext(shareID, bytes.Repeat([]byte{9}, 32), 5),
		"epoch":  share.GetSealContext(shareID, reader, 6),
	} {
		if bytes.Equal(other, got) {
			t.Errorf("a change of the %s leaves the context as it was", name)
		}
	}
}

func TestGetOwnerBodyContext_FollowsTheFrozenLayout(t *testing.T) {
	t.Parallel()
	shareID := bytes.Repeat([]byte{1}, 16)

	got := share.GetOwnerBodyContext(shareID, 258)

	want := append(append([]byte("numen-sync/share-owner/v1"), shareID...), 0, 0, 0, 0, 0, 0, 1, 2)
	if !bytes.Equal(got, want) {
		t.Errorf("context = %x, want %x", got, want)
	}
}

func TestReaderCode_CarriesTheChainOfTheReaderAndNoSecret(t *testing.T) {
	t.Parallel()
	chain := newChain(t)

	code := share.NewReaderCode(chain.Bytes())
	parsed, err := share.ParseReaderCode(code)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(code, "numen-reader-1.") || strings.ContainsAny(strings.TrimPrefix(code, "numen-reader-1."), "+/= ") {
		t.Errorf("code = %q, want the prefix and base64url without padding", code)
	}
	if !bytes.Equal(parsed.Bytes(), chain.Bytes()) || parsed.Root().Fingerprint() != chain.Root().Fingerprint() {
		t.Error("the parsed chain differs from the chain of the code")
	}
}

func TestParseReaderCode_RefusesWhatIsNotACodeOfAVerifiedChain(t *testing.T) {
	t.Parallel()
	chain := newChain(t)
	damaged := chain.Bytes()
	damaged[len(damaged)-1] ^= 1
	for name, text := range map[string]string{
		"empty":           "",
		"no prefix":       strings.TrimPrefix(share.NewReaderCode(chain.Bytes()), "numen-reader-1."),
		"an invitation":   "numen-share-1.AAAA",
		"not base64":      "numen-reader-1.!!!",
		"not a chain":     "numen-reader-1.AAAA",
		"a damaged chain": share.NewReaderCode(damaged),
		"spaces around":   " " + share.NewReaderCode(chain.Bytes()) + " x",
	} {
		if _, err := share.ParseReaderCode(text); err == nil {
			t.Errorf("%s: the text was accepted", name)
		}
	}
}

func TestInvitation_ContainsShareReaderFingerprintAndOwnerChain(t *testing.T) {
	t.Parallel()
	owner, reader := newChain(t), newChain(t)
	want := share.Invitation{ShareRootID: bytes.Repeat([]byte{4}, 16), ReaderFingerprint: reader.Root().Fingerprint(), OwnerChain: owner.Bytes()}

	text := share.NewInvitation(want)
	got, err := share.ParseInvitation(text)

	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(text, "numen-share-1.") || !bytes.Equal(got.ShareRootID, want.ShareRootID) ||
		got.ReaderFingerprint != want.ReaderFingerprint || !bytes.Equal(got.OwnerChain, want.OwnerChain) {
		t.Errorf("invitation = %+v from %q, want %+v", got, text, want)
	}
}

func TestParseInvitation_RefusesATextThatIsNotAnInvitationOrIsTooShort(t *testing.T) {
	t.Parallel()
	for name, text := range map[string]string{
		"empty":         "",
		"no prefix":     "AAAA",
		"a reader code": share.NewReaderCode(newChain(t).Bytes()),
		"not base64":    "numen-share-1.!!!",
		"too short":     "numen-share-1.AAAA",
	} {
		if _, err := share.ParseInvitation(text); !errors.Is(err, share.ErrBadCode) {
			t.Errorf("%s: error = %v, want ErrBadCode", name, err)
		}
	}
}

func TestGetRelativePath_TheVaultRootHoldsEveryFile(t *testing.T) {
	t.Parallel()
	for _, root := range []string{"", ".", "/"} {
		got, ok := share.GetRelativePath(root, "Food/Soup.md")
		if !ok || got != "Food/Soup.md" {
			t.Errorf("root %q: got %q (%v), want the whole path", root, got, ok)
		}
	}
}

func TestCheckNesting_NamesTheShareInTheWay(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		folder string
		shared []string
		want   string
	}{
		"the vault holds a folder":    {folder: "", shared: []string{"Recipes"}, want: "the vault cannot be shared while its folder «Recipes» is shared"},
		"a folder holds a folder":     {folder: "Life", shared: []string{"Life/Food"}, want: "«Life» cannot be shared while its folder «Life/Food» is shared"},
		"a folder inside a folder":    {folder: "Food/Dinner", shared: []string{"Food"}, want: "«Food/Dinner» is inside the shared folder «Food»"},
		"a folder inside the vault":   {folder: "Food", shared: []string{""}, want: "«Food» is inside the shared vault"},
		"the vault is shared already": {folder: "", shared: []string{""}, want: "the vault is shared already"},
	} {
		if err := share.CheckNesting(c.folder, c.shared); err == nil || err.Error() != c.want {
			t.Errorf("%s: error = %v, want %q", name, err, c.want)
		}
	}
}
