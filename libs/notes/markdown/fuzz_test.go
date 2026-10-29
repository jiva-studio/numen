package markdown

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// spliceSeeds are notes a write has to come through byte for byte: a document
// end standing inside the delimiters, a block scalar carrying a line that opens
// with `#`, a list written flush with its key, keys and values written in other
// scripts, and the plain shapes around them.
var spliceSeeds = []string{
	"---\ntitle: Old\n---\nbody\n",
	"---\ntitle: Old\n...\nmy own scratch notes, not yaml: [[[\nkeep me\n---\nbody\n",
	"---\n# a note to myself\ntitle: Old\n\nzebra: 1\n---\nbody\n",
	"---\ntitle:\n- one\n- two\nzebra: 1\n---\nbody\n",
	"---\nlinks:\n  - to: A\n    role: ref\n    note: |\n      the first thought\n      # the second one\n  - to: B\n    role: ref\n---\nbody\n",
	"---\nзаголовок: Мхи\nlinks:\n  - to: Папоротники\n    role: ref\n---\nтекст\n",
	"---\nτίτλος: Βρύα\nlinks:\n  - to: \"[[Φτέρες]]\"\n    role: ref\n---\nκείμενο\n",
	"---\n标题: 苔藓\nlinks:\n  - to: 蕨类\n    role: ref\n    note: 一\n---\n正文\n",
	"---\nё中a: a\nlinks:\n  - to: a\n    role: ref\n---\nbody\n",
	"---\nзаметка: |\n  первая мысль\n  вторая мысль\ntitle: Мхи\n---\nтекст\n",
	"---\n0: \n... 0\n---\nbody\n",
	"---\n 0:\n0\n---\nbody\n",
	"---\r\ntitle: Old\r\n---\r\nbody\r\n",
	"---\ntitle: Old\n---",
	"---\ntitle: Old\nnever closed\n",
	"# no frontmatter at all\n",
	"",
}

// A write to one key leaves every byte outside that key unchanged.
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
		checkWriteInsideKey(t, raw, string(d.Bytes()), "title")
	})
}

// Writing a link leaves every byte outside the `links:` block unchanged.
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
		checkWriteInsideKey(t, raw, string(d.Bytes()), "links")
	})
}

// A change to a link stays inside the `links:` block, and a rejected one leaves
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
				t.Fatalf("a rejected change was written\n was %q\n now %q", was, got)
			}
			return
		}
		checkWriteInsideKey(t, was, string(d.Bytes()), "links")
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
		head := string(d.bom) + string(d.open) + string(d.front) + string(d.closeDelimiter)
		if err := d.SetBody(body); err != nil {
			return
		}
		if got := string(d.Bytes()); !strings.HasPrefix(got, head) {
			t.Fatalf("writing the prose changed the frontmatter\n want %q\n  got %q", head, got)
		}
	})
}

// checkWriteInsideKey fails unless the bytes a write replaced all belong to the
// lines one top-level key owns.
func checkWriteInsideKey(t *testing.T, before, after, key string) {
	t.Helper()
	from, to := getChangedRun(before, after)
	if from == to {
		return
	}
	start, end, found := getOwnedLines(before, key)
	if !found {
		return
	}
	if from < start || to > end {
		t.Fatalf("a write took bytes %d:%d, and %s owns %d:%d\n was %q\n now %q",
			from, to, key, start, end, before, after)
	}
}

// getChangedRun returns the range of the note that a write replaced: the bytes
// after the common prefix of both versions and before their common suffix.
func getChangedRun(before, after string) (int, int) {
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

// getOwnedLines returns the widest range of frontmatter that one top-level key
// owns: the key's own line, and the following lines that are indented under it
// or continue it as a list. It reports found=false when the key is absent.
func getOwnedLines(note, key string) (start, end int, found bool) {
	var starts []int
	var lines []string
	for pos := 0; pos < len(note); {
		stop := len(note)
		if next := strings.IndexByte(note[pos:], '\n'); next >= 0 {
			stop = pos + next + 1
		}
		starts = append(starts, pos)
		lines = append(lines, strings.TrimRight(note[pos:stop], "\r\n"))
		pos = stop
	}
	starts = append(starts, len(note))
	if len(lines) == 0 || lines[0] != "---" {
		return 0, 0, false
	}
	closingLine := 0
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			closingLine = i
			break
		}
	}
	if closingLine == 0 {
		return 0, 0, false
	}

	margin, foundMargin, owner := "", false, 0
	for i := 1; i < closingLine; i++ {
		text := strings.TrimSpace(lines[i])
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		if !foundMargin {
			margin, foundMargin = getIndent(lines[i]), true
		}
		if len(getIndent(lines[i])) == len(margin) && strings.HasPrefix(text, key+":") {
			owner = i
			break
		}
	}
	if owner == 0 {
		return 0, 0, false
	}

	last := owner
	for i := owner + 1; i < closingLine; i++ {
		text := strings.TrimSpace(lines[i])
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		if len(getIndent(lines[i])) <= len(margin) && !strings.HasPrefix(text, "-") {
			break
		}
		last = i
	}
	return starts[owner], starts[last+1], true
}
