import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { after, describe, it } from 'node:test'
import { fileURLToPath } from 'node:url'

const SCAN = path.join(path.dirname(fileURLToPath(import.meta.url)), 'scan.mjs')
const ROOT = fs.mkdtempSync(path.join(os.tmpdir(), 'lint-narrative-'))
after(() => fs.rmSync(ROOT, { recursive: true, force: true }))

/** Writes one fixture file and runs the scan over it, as `--check` does in CI. */
function scanFixture(name, content) {
  const file = path.join(ROOT, name)
  fs.mkdirSync(path.dirname(file), { recursive: true })
  fs.writeFileSync(file, content)
  const run = spawnSync('node', [SCAN, file, '--check'], { encoding: 'utf8' })
  return { code: run.status, out: run.stdout }
}

function assertFlagged(name, content, category) {
  const { code, out } = scanFixture(name, content)
  assert.equal(code, 1, `${name} must fail the check`)
  assert.match(out, new RegExp(`\\[${category}`), `${name} must be reported under ${category}`)
}

function assertClean(name, content) {
  const { code, out } = scanFixture(name, content)
  assert.equal(code, 0, `${name} must pass the check:\n${out}`)
}

describe('Markdown documents', () => {
  it('flags a window that wears a theme', () => {
    assertFlagged('a/themes.md', '# Themes\n\nThe window wears the theme from the first frame.\n', 'proseDocs')
  })
  it('flags a heading with wearing in it', () => {
    assertFlagged('b/themes.md', '## The window opens wearing it\n', 'proseDocs')
  })
  it('flags hands over, speaks and brings the index level', () => {
    assertFlagged('c/a.md', 'A page hands over what it holds.\n', 'proseDocs')
    assertFlagged('c/b.md', 'The host speaks protocol 1.\n', 'proseDocs')
    assertFlagged('c/c.md', 'A write brings the index level with the file.\n', 'proseDocs')
  })
  it('flags stands and sits at a place', () => {
    assertFlagged('d/a.md', 'The value stands under the key it names.\n', 'proseDocs')
  })
  it('accepts plain wording', () => {
    assertClean('e/themes.md', 'The window applies the theme from the first frame.\n\nA page passes what it holds.\n')
  })
  it('accepts a banned word quoted in code font or in a fenced block', () => {
    assertClean('f/rules.md', 'Write `applies`, not `wears`.\n\n```\nthe window wears a theme\n```\n')
  })
  it('accepts words that only contain a banned one', () => {
    assertClean('g/address.md', 'The address of the dressing room is not a theme. A forward is not worn out.\n'.replace('dressing room', 'address book').replace('worn out', 'sworn'))
  })
})

describe('comments', () => {
  it('flags a comment with a personification', () => {
    assertFlagged('h/a.go', 'package a\n\n// The window wears a theme.\nfunc f() {}\n', 'anglishComments')
    assertFlagged('h/b.ts', '// An interface with one method is a struct wearing a costume.\nexport const b = 1\n', 'anglishComments')
  })
  it('accepts a plain comment', () => {
    assertClean('i/a.go', 'package a\n\n// The window applies a theme.\nfunc f() {}\n')
  })
})

describe('strings a person reads', () => {
  it('flags a Go task title', () => {
    assertFlagged('j/task.go', 'package a\n\nvar task = run.Task{ID: "theme", Title: "Wearing a theme"}\n', 'uiStrings')
  })
  it('flags a Go error text', () => {
    assertFlagged('k/err.go', 'package a\n\nvar err = errors.New("plugin x speaks protocol 2")\n', 'uiStrings')
  })
  it('flags a words file entry', () => {
    assertFlagged('l/sampleWords.ts', "export const words = {\n  theme: 'Wearing a theme',\n}\n", 'ui-words|uiWords')
  })
  it('flags a manifest title, description and detail', () => {
    for (const key of ['title', 'description', 'detail']) {
      assertFlagged(`m-${key}/manifest.json`, JSON.stringify({ settings: [{ [key]: 'Wears the theme' }] }), 'uiStrings')
    }
  })
  it('accepts imperative task titles and manifest texts', () => {
    assertClean('n/task.go', 'package a\n\nvar task = run.Task{ID: "theme", Title: "Apply theme", Error: "Theme not found: dracula"}\n')
    assertClean('o/manifest.json', JSON.stringify({ commands: [{ title: 'Apply theme', description: 'Sets the theme of the window' }] }))
  })
  it('does not read a manifest key that is not a text', () => {
    assertClean('p/manifest.json', JSON.stringify({ id: 'wears', kind: 'wears' }))
  })
})

describe('test titles', () => {
  it('flags a TypeScript test title', () => {
    assertFlagged('q/a.test.ts', "it('wears the theme the keyboard walks onto', () => {})\n", 'narrativeTestTitles')
  })
  it('flags a Go test name', () => {
    assertFlagged('r/a_test.go', 'package a\n\nfunc TestThePageOpensWearingTheTheme(t *testing.T) {}\n', 'narrativeTestTitles')
  })
  it('flags a Go test message', () => {
    assertFlagged('s/b_test.go', 'package a\n\nfunc TestTheme(t *testing.T) {\n\tt.Errorf("wears %q", name)\n}\n', 'uiStrings')
  })
  it('accepts fixture strings in tests that are not messages', () => {
    assertClean('t/c_test.go', 'package a\n\nfunc TestTheme(t *testing.T) {\n\tname := "A whale breathes air."\n\t_ = name\n}\n')
  })
})

describe('skipped paths', () => {
  it('does not scan apps/landing while the website is being rewritten', () => {
    const file = path.join(ROOT, 'repo/apps/landing/page.md')
    fs.mkdirSync(path.dirname(file), { recursive: true })
    fs.writeFileSync(file, 'The window wears a theme.\n')
    const run = spawnSync('node', [SCAN, path.join(ROOT, 'repo'), '--check'], { encoding: 'utf8' })
    assert.equal(run.status, 0, run.stdout)
  })
})
