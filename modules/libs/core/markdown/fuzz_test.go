package markdown

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// spliceSeeds are notes a write has to come through byte for byte: a document
// end standing inside the delimiters, a block scalar carrying a line that opens
// with `#`, a list written flush with its key, and the plain shapes around them.
var spliceSeeds = []string{
	"---\ntitle: Old\n---\nbody\n",
	"---\ntitle: Old\n...\nmy own scratch notes, not yaml: [[[\nkeep me\n---\nbody\n",
	"---\n# a note to myself\ntitle: Old\n\nzebra: 1\n---\nbody\n",
	"---\ntitle:\n- one\n- two\nzebra: 1\n---\nbody\n",
	"---\nlinks:\n  - to: A\n    role: ref\n    note: |\n      the first thought\n      # the second one\n  - to: B\n    role: ref\n---\nbody\n",
	"---\n0: \n... 0\n---\nbody\n",
	"---\n 0:\n0\n---\nbody\n",
	"---\r\ntitle: Old\r\n---\r\nbody\r\n",
	"---\ntitle: Old\n---",
	"---\ntitle: Old\nnever closed\n",
	"# no frontmatter at all\n",
	"",
}

// A write to one key leaves every byte the key does not own where it stood.
func FuzzSetTitle(f *testing.F) {
	for _, seed := range spliceSeeds {
		f.Add(seed, "New")
	}
	f.Fuzz(func(t *testing.T, raw, title string) {
		d, err := Open([]byte(raw))
		if err != nil {
			return
		}
		if err := d.SetTitle(title); err != nil {
			return
		}
		keptOutside(t, raw, string(d.Bytes()), "title")
	})
}

// Writing a link leaves every byte outside the `links:` block where it stood.
func FuzzAddLink(f *testing.F) {
	for _, seed := range spliceSeeds {
		f.Add(seed, "Somewhere Else")
	}
	f.Fuzz(func(t *testing.T, raw, to string) {
		d, err := Open([]byte(raw))
		if err != nil {
			return
		}
		if err := d.AddLink(domain.Link{
			Target: domain.ParseAddress(to), Role: domain.RoleRef,
		}); err != nil {
			return
		}
		keptOutside(t, raw, string(d.Bytes()), "links")
	})
}

// A change to a link stays inside the `links:` block, and a refused one leaves
// the note byte for byte.
func FuzzUpdateLink(f *testing.F) {
	for _, seed := range spliceSeeds {
		f.Add(seed, "A", "child")
	}
	f.Add("---\nlinks:\n  - to: A\n    role: ref\n    mine: keep me\n"+
		"  - to: A\n    role: ref\n---\nbody\n", "A", "child")
	f.Add("---\nlinks:\n  - to: A\n    role: ref\n"+
		"  - to: A\n    role: ref\n    mine: keep me\n---\nbody\n", "A", "child")
	f.Add("---\nlinks:\r- to: 0\n0:\n---", "0", "child")
	f.Fuzz(func(t *testing.T, raw, to, role string) {
		d, err := Open([]byte(raw))
		if err != nil {
			return
		}
		was := string(d.Bytes())
		if _, err := d.UpdateLink(domain.ParseAddress(to), domain.Link{Role: domain.LinkRole(role)}); err != nil {
			if got := string(d.Bytes()); got != was {
				t.Fatalf("a refused change was written\n was %q\n now %q", was, got)
			}
			return
		}
		keptOutside(t, was, string(d.Bytes()), "links")
	})
}

// Writing the prose leaves the frontmatter byte for byte.
func FuzzSetBody(f *testing.F) {
	for _, seed := range spliceSeeds {
		f.Add(seed, "# A heading\n\nAnd a sentence.\n")
	}
	f.Fuzz(func(t *testing.T, raw, body string) {
		d, err := Open([]byte(raw))
		if err != nil {
			return
		}
		head := string(d.bom) + string(d.open) + string(d.front) + string(d.shut)
		d.SetBody(body)
		if got := string(d.Bytes()); !strings.HasPrefix(got, head) {
			t.Fatalf("writing the prose changed the frontmatter\n want %q\n  got %q", head, got)
		}
	})
}

// keptOutside fails unless the bytes a write replaced all belong to the lines
// one top-level key owns.
func keptOutside(t *testing.T, before, after, key string) {
	t.Helper()
	from, to := changed(before, after)
	if from == to {
		return
	}
	start, end, found := ownedLines(before, key)
	if !found {
		return
	}
	if from < start || to > end {
		t.Fatalf("a write took bytes %d:%d, and %s owns %d:%d\n was %q\n now %q",
			from, to, key, start, end, before, after)
	}
}

// changed is the run of the note a write replaced: what stands after the bytes
// both spellings open with, and before the bytes both close with.
func changed(before, after string) (int, int) {
	from := 0
	for from < len(before) && from < len(after) && before[from] == after[from] {
		from++
	}
	to := len(before)
	for to > from {
		mirror := len(after) - (len(before) - to) - 1
		if mirror < from || before[to-1] != after[mirror] {
			break
		}
		to--
	}
	return from, to
}

// ownedLines is the widest run of frontmatter one top-level key can be said to
// own: the key's own line, and the lines below it standing indented under it or
// hanging from it as a list. It says nothing of a key it cannot find.
func ownedLines(note, key string) (start, end int, found bool) {
	var starts []int
	var lines []string
	for at := 0; at < len(note); {
		stop := len(note)
		if next := strings.IndexByte(note[at:], '\n'); next >= 0 {
			stop = at + next + 1
		}
		starts = append(starts, at)
		lines = append(lines, strings.TrimRight(note[at:stop], "\r\n"))
		at = stop
	}
	starts = append(starts, len(note))
	if len(lines) == 0 || lines[0] != "---" {
		return 0, 0, false
	}
	shut := 0
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			shut = i
			break
		}
	}
	if shut == 0 {
		return 0, 0, false
	}

	margin, at, owner := "", false, 0
	for i := 1; i < shut; i++ {
		text := strings.TrimSpace(lines[i])
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		if !at {
			margin, at = leading(lines[i]), true
		}
		if len(leading(lines[i])) == len(margin) && strings.HasPrefix(text, key+":") {
			owner = i
			break
		}
	}
	if owner == 0 {
		return 0, 0, false
	}

	last := owner
	for i := owner + 1; i < shut; i++ {
		text := strings.TrimSpace(lines[i])
		if text == "" {
			continue
		}
		if len(leading(lines[i])) <= len(margin) && !strings.HasPrefix(text, "-") {
			break
		}
		last = i
	}
	return starts[owner], starts[last+1], true
}
