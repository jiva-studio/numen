import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { after, describe, it } from 'node:test'
import { fileURLToPath } from 'node:url'

const SCAN = path.join(path.dirname(fileURLToPath(import.meta.url)), 'check.mjs')
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
  it('flags a personified or theatrical phrase', () => {
    assertFlagged('i/perf.md', 'Repeating it measures patience.\n', 'proseDocs')
    assertFlagged('i/tabs.md', 'The list hangs under the heading and the page opens of its own accord.\n', 'proseDocs')
    assertFlagged('i/core.md', 'The core knows nothing about proofreading.\n', 'proseDocs')
    assertFlagged('i/ui.md', 'A plugin that works seamlessly takes you to the download.\n', 'proseDocs')
  })
  it('flags the ground and the door as figures', () => {
    assertFlagged('j/theme.md', 'The ground under code is darker.\n', 'proseDocs')
    assertFlagged('j/off.md', '## Shutting the door\n', 'proseDocs')
  })
  it('accepts the plain wording of the same facts', () => {
    assertClean('k/plain.md', 'The list is placed under the heading and the page opens automatically.\n\nThe core does not depend on proofreading. The background of code is darker.\n')
  })
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

describe('function names', () => {
  it('flags a Go function named with a third-person verb', () => {
    const { code, out } = scanFixture('fn/a.go', 'package p\n\nfunc carries() {}\n')
    assert.equal(code, 1)
    assert.match(out, /3rd-person singular functions[^\n]*: 1/)
  })
  it('flags a TypeScript function named with a third-person verb', () => {
    const { code, out } = scanFixture('fn/a.ts', 'export function reads(): void {}\n')
    assert.equal(code, 1)
    assert.match(out, /3rd-person singular functions[^\n]*: 1/)
  })
  it('accepts an imperative function name and leaves the names of variables to the reader', () => {
    assertClean('fn/b.go', 'package p\n\nfunc getVault() {}\n\nfunc use() {\n\theld := 1\n\t_ = held\n}\n')
  })
})

describe('fake Turn values', () => {
  it('flags a held key after a brace, a comma or two spaces', () => {
    assert.match(scanFixture('u/a.ts', 'const a = { held: x }\n').out, /key held:/)
    assert.match(scanFixture('u/b.ts', 'const a = f(1,held: x)\n').out, /key held:/)
    assert.match(scanFixture('u/c.ts', 'const a = f(1)   refuses: x\n').out, /key refuses:/)
  })
  it('does not flag a held key after a single space', () => {
    assert.doesNotMatch(scanFixture('u/d.ts', 'const a = b ? held: x\n').out, /key held:/)
  })
})

describe('--fix', () => {
  it('removes a whole-line comment with a listed phrase and keeps other comments', () => {
    const file = path.join(ROOT, 'v/poetic.ts')
    fs.mkdirSync(path.dirname(file), { recursive: true })
    fs.writeFileSync(file, 'const a = 1\n  // The page Follows  the hand.\n// Plain comment.\nconst b = 2\n')
    spawnSync('node', [SCAN, file, '--fix'], { encoding: 'utf8' })
    assert.equal(fs.readFileSync(file, 'utf8'), 'const a = 1\n// Plain comment.\nconst b = 2\n')
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

/** The phrase rules (category 24) that a fixture file breaks, one rule name per finding. */
function getPhraseRules(name, content) {
  const file = path.join(ROOT, name)
  const report = path.join(ROOT, `${name}.report.json`)
  fs.mkdirSync(path.dirname(file), { recursive: true })
  fs.writeFileSync(file, content)
  const run = spawnSync('node', [SCAN, file, '--category', 'phrases', '--json', report], { encoding: 'utf8' })
  assert.equal(run.status, 0, run.stderr)
  return JSON.parse(fs.readFileSync(report, 'utf8')).narrativePhrases.map((item) => item.word)
}

describe('narrative phrases', () => {
  const badTitles = [
    ['lets go of the interval it is waiting on', 'let go'],
    ['is told, left out of the layout, and lets go of what it started', 'let go'],
    ['stays where it is when one item is all there is to be on', 'stays where'],
    ['ends a cancelled drag where nobody requested to hear of it', 'hears'],
    ['releases the pointer capture when the scope goes', 'goes/leaves'],
    ['reports a report the core did not take', 'does not take'],
    ['returns the value that comes back from the core', 'comes back'],
    ['is told the width of the box', 'is told'],
    ['brings along the tab state', 'brings along'],
    ['opens no menu where nobody is listening', 'nobody'],
  ]
  for (const [title, rule] of badTitles) {
    it(`reports the TypeScript test title "${title}" under ${rule}`, () => {
      const rules = getPhraseRules(`u/${rule.replace(/\W/g, '')}-${title.length}.test.ts`, `it('${title}', () => {})\n`)
      assert.deepEqual(rules, [rule])
    })
  }

  it('reports narrative comments', () => {
    const content = [
      '/** A point in the window, put on the screen. */',
      '/** What letting go after a drag evaluates to. */',
      '// isOpen answers whether the panel is open.',
      '// The list keeps what it holds after a reload.',
      'export const a = 1',
      '',
    ].join('\n')
    assert.deepEqual(getPhraseRules('v/comments.ts', content), ['put on', 'let go', 'answers whether', 'what it holds'])
  })

  it('reports a Go test name split on case and a t.Run name', () => {
    const content = [
      'package a',
      '',
      'func TestTheCoreLetsGoOfTheLock(t *testing.T) {',
      '\tt.Run("is told the size", func(t *testing.T) {})',
      '}',
      '',
    ].join('\n')
    assert.deepEqual(getPhraseRules('w/a_test.go', content), ['let go', 'is told'])
  })

  it('accepts the plain forms', () => {
    const ts = [
      '/** Converts a viewport point to screen coordinates. */',
      '// isOpen returns whether the panel is open.',
      '// After the pointer leaves, the title returns to its default.',
      "it('releases pointer capture when the scope is disposed', () => {})",
      "it('reports an error when the core rejects the tab report', () => {})",
      "it('keeps its position when the list has one item', () => {})",
      "it('lets Go code call the bridge', () => {})",
      '',
    ].join('\n')
    assert.deepEqual(getPhraseRules('x/plain.test.ts', ts), [])
    const go = [
      'package a',
      '',
      '// Lock holds the mutex for the write.',
      '// The caller holds it until Close returns; the output goes to stderr.',
      'func TestLockReleasesTheMutex(t *testing.T) {',
      '\tt.Run("returns the stored value", func(t *testing.T) {})',
      '}',
      '',
    ].join('\n')
    assert.deepEqual(getPhraseRules('x/plain_test.go', go), [])
  })

  it('fails on a phrase in a full scan', () => {
    const { code, out } = scanFixture('y/a.test.ts', "it('lets go of the interval it is waiting on', () => {})\n")
    assert.equal(code, 1, out)
    assert.match(out, /24\. Narrative phrases[^:]*: 1/)
  })
})

describe('diff mode', () => {
  const git = (repo, ...args) => {
    const run = spawnSync('git', ['-c', 'user.name=test', '-c', 'user.email=test@example.com', '-c', 'commit.gpgsign=false', ...args], { cwd: repo, encoding: 'utf8' })
    assert.equal(run.status, 0, run.stderr)
    return run.stdout
  }
  const OLD_LINE = "it('lets go of the interval it is waiting on', () => {})"
  const NEW_LINE = "it('releases the pointer capture when the scope goes', () => {})"

  /** A repository with one commit holding OLD_LINE in a.test.ts. */
  function createRepo(name) {
    const repo = path.join(ROOT, name)
    fs.mkdirSync(repo, { recursive: true })
    git(repo, 'init', '-q', '-b', 'main')
    fs.writeFileSync(path.join(repo, 'a.test.ts'), `${OLD_LINE}\n`)
    git(repo, 'add', '.')
    git(repo, 'commit', '-q', '-m', 'base')
    return repo
  }

  it('--changed-since reports only the phrases on lines added since the merge base', () => {
    const repo = createRepo('diff-changed')
    git(repo, 'checkout', '-q', '-b', 'topic')
    fs.writeFileSync(path.join(repo, 'a.test.ts'), `${OLD_LINE}\n${NEW_LINE}\n`)
    git(repo, 'commit', '-q', '-am', 'add a title')
    const run = spawnSync('node', [SCAN, repo, '--check', '--changed-since', 'main'], { encoding: 'utf8' })
    assert.equal(run.status, 1, run.stdout)
    assert.match(run.stdout, /a\.test\.ts:2 \[narrativePhrases: goes\/leaves\]/)
    assert.doesNotMatch(run.stdout, /a\.test\.ts:1 /)
  })

  it('--changed-since passes when the added lines are plain', () => {
    const repo = createRepo('diff-plain')
    git(repo, 'checkout', '-q', '-b', 'topic')
    fs.writeFileSync(path.join(repo, 'a.test.ts'), `${OLD_LINE}\nit('releases pointer capture when the scope is disposed', () => {})\n`)
    git(repo, 'commit', '-q', '-am', 'add a title')
    const run = spawnSync('node', [SCAN, repo, '--check', '--changed-since', 'main'], { encoding: 'utf8' })
    assert.equal(run.status, 0, run.stdout)
  })

  it('--changed-since exits with code 2 on a ref git does not know', () => {
    const repo = createRepo('diff-unknown')
    const run = spawnSync('node', [SCAN, repo, '--check', '--changed-since', 'no-such-ref'], { encoding: 'utf8' })
    assert.equal(run.status, 2, run.stdout)
  })

  it('--staged reports only the phrases on staged added lines', () => {
    const repo = createRepo('diff-staged')
    fs.writeFileSync(path.join(repo, 'a.test.ts'), `${OLD_LINE}\n${NEW_LINE}\n`)
    git(repo, 'add', 'a.test.ts')
    const run = spawnSync('node', [SCAN, repo, '--staged', '--check'], { encoding: 'utf8' })
    assert.equal(run.status, 1, run.stdout)
    assert.match(run.stdout, /a\.test\.ts:2 \[narrativePhrases: goes\/leaves\]/)
    assert.doesNotMatch(run.stdout, /a\.test\.ts:1 /)
  })
})

describe('function names', () => {
  it('flags a third-person function name in Go and TypeScript', () => {
    assertFlagged('f/a.go', 'package a\n\nfunc reads() {}\n', 'thirdPersonFunctions')
    assertFlagged('f/a.ts', 'export function opens() {}\n', 'thirdPersonFunctions')
  })
  it('accepts an imperative function name', () => {
    assertClean('f/b.go', 'package b\n\nfunc readVault() {}\n')
    assertClean('f/b.ts', 'export function openVault() {}\n')
  })
})
