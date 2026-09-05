import assert from 'node:assert/strict'
import test from 'node:test'
import { calls, carries, goDeclares, holds, named, refused, stemOf, tsDeclares } from './filenames.mjs'

/**
 * A name is answered for by the code under it or it is a word standing on
 * nothing. `owing_test.go` said "owing" nowhere in itself — what it holds is
 * the counting of a vault's cards due — and no rule in this repository could
 * see it, because every one of them read declarations and none read the name
 * of the file they stand in.
 */
test('every file named by a verb form says that word in its own code', () => {
  const found = named()
  const wrong = found
    .filter(({ wrong }) => wrong.length > 0)
    .map(({ at, wrong }) => `${at}: nothing here is called ${wrong.join(' or ')}`)
  assert.deepEqual(wrong, [])

  // A walk that read no file is a rule checked against nothing, and it passes.
  // A count cannot say which files it read, so one file of each kind it reads
  // is named, in the modules furthest apart: a rule stops at a border first.
  assert.ok(found.length > 1200, `${found.length} files read: the walk is not reading the modules`)
  for (const one of [
    'modules/libs/core/domain/card.go',
    'modules/apps/desktop/cmd/numen/main.go',
    'modules/apps/mobile/bind/mobile.go',
    'modules/libs/ui/src/digits.ts',
    'modules/libs/ui/src/welcome/letters.ts',
    'modules/libs/ui/src/notices/LiveRegions.vue',
    'modules/apps/mobile/src/core.ts',
  ]) {
    assert.ok(
      found.some(({ at }) => at === one),
      `the walk did not read ${one}, so the rule stops before it`,
    )
  }
})

/**
 * What the rule refuses, read against names written to be refused. A word is
 * answered by the same word in another tense of the reader's — naming by Name,
 * placed by place — and not by another tense of its own: Attended is a second
 * name for the doing of a thing, and says no more about what a file holds than
 * attending does.
 */
test('what the file-name rule refuses', () => {
  const cases = [
    { says: 'a gerund nothing is called', allowed: false, stem: 'owing', names: ['WatchCardsDue'] },
    { says: 'a participle nothing is called', allowed: false, stem: 'stored', names: ['Look'] },
    { says: 'a gerund answered only by a participle', allowed: false, stem: 'attending', names: ['Attended'] },
    { says: 'a gerund behind another word', allowed: false, stem: 'serving_reading', names: ['Read'] },
    { says: 'a gerund behind a component', allowed: false, stem: 'App.opening', names: ['App'] },
    { says: 'a gerund the file declares', allowed: true, stem: 'reading', names: ['Reading'] },
    { says: 'a gerund whose verb the file declares', allowed: true, stem: 'naming', names: ['NameSource'] },
    { says: 'a participle whose verb the file declares', allowed: true, stem: 'placed', names: ['place'] },
    { says: 'a gerund the package is named after', allowed: true, stem: 'chunking', names: ['chunking'] },
    { says: 'a gerund a composable is named after', allowed: true, stem: 'placing', names: ['usePlace'] },
    { says: 'a plural of what the file declares', allowed: true, stem: 'weighed', names: ['weighs'] },
    { says: 'a name that is no verb form', allowed: true, stem: 'vault', names: ['Nothing'] },
    { says: 'a build tag the file is not named for', allowed: true, stem: 'rename_windows', names: ['Rename'] },
  ]

  const wrong = cases.filter((one) => refused(one.stem, one.names).length > 0).map((one) => one.says)
  const wanted = cases.filter((one) => !one.allowed).map((one) => one.says)
  assert.deepEqual(wrong, wanted)
})

/** The names the reading has to find, and what it must not read as one. */
test('what a file says its own names are', () => {
  const source = [
    'package alpha',
    'type Beta struct{ gamma int }',
    'func (b Beta) Delta() {}',
    'func epsilon() { zeta := 1 }',
    'var Eta = 2',
    'const (',
    '\tTheta = 3',
    ')',
    '// Iota is named only in a comment',
  ].join('\n')
  assert.deepEqual(goDeclares(source), ['alpha', 'Beta', 'Delta', 'epsilon', 'Eta', 'Theta'])

  const module = [
    "import { Kappa } from './lambda'",
    'export interface Mu {}',
    'export const nu = 1',
    'export default function xi() {',
    '  const omicron = 2',
    '}',
    'type Pi = string',
  ].join('\n')
  assert.deepEqual(tsDeclares(module), ['Mu', 'nu', 'xi', 'Pi'])

  const test = ['func TestKappa(t *testing.T) {', '\tlambda(mu)', '\t_ = "Nu"', '\t// Xi', '}'].join('\n')
  assert.deepEqual(calls(test), ['func', 'TestKappa', 't', 'testing', 'T', 'lambda', 'mu', '_'])
})

/**
 * A single-file component is addressed by its file name and declares itself by
 * standing there, which is why `RecordingTab.vue` and `SettingRow.vue` are no
 * business of this rule. Nothing else is read that way.
 */
test('what a component declares', () => {
  const component = [
    '<script setup lang="ts">',
    'const props = defineProps<{ at: number }>()',
    '</script>',
    '<template><p>{{ props.at }}</p></template>',
  ].join('\n')
  assert.deepEqual(holds({ at: 'a/b/RecordingTab.vue', text: component }), ['RecordingTab', 'props'])
  assert.deepEqual(refused('RecordingTab', holds({ at: 'a/b/RecordingTab.vue', text: component })), [])
  assert.deepEqual(holds({ at: 'a/b/counting.ts', text: 'export const grouped = (n) => n' }), ['grouped'])
})

/** What a file's name is, once the endings a tool or a suite put there are off. */
test('the stem of a file name', () => {
  assert.deepEqual(stemOf('a/b/reading_test.go'), { stem: 'reading', test: true })
  assert.deepEqual(stemOf('a/b/reading_bench_test.go'), { stem: 'reading_bench', test: true })
  assert.deepEqual(stemOf('a/b/reading.go'), { stem: 'reading', test: false })
  assert.deepEqual(stemOf('a/b/reading.test.ts'), { stem: 'reading', test: true })
  assert.deepEqual(stemOf('a/b/Reader.stories.ts'), { stem: 'Reader', test: true })
  assert.deepEqual(stemOf('a/b/reading.ts'), { stem: 'reading', test: false })
  assert.deepEqual(stemOf('a/b/Reader.vue'), { stem: 'Reader', test: false })
  assert.ok(carries('naming', 'names'))
  assert.ok(!carries('naming', 'named'))
})
