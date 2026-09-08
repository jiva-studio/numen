import assert from 'node:assert/strict'
import test from 'node:test'
import { fixturesOf, refused, walked } from './exports.mjs'

/** The barrel the hand-written cases below are read against. */
const SURFACE = new Map([['@numen/ui', 'modules/libs/ui/src/index.ts']])

/** One case's findings, as `file::name`, in the order the rule reports them. */
const found = (files, surface = SURFACE) =>
  refused(files, surface)
    .map(({ at, name }) => `${at}::${name}`)
    .sort()

/**
 * A test is not a use. `limiting` stood in the notes window's preset for six
 * days after its last caller went, and was refactored twice in that time
 * because its test still imported it and still passed — the compiler saw an
 * export used, and no rule here looked past the file a name was declared in.
 * The one assertion left about it looked for text no code could produce, so
 * what held the function up was a test that could not fail.
 */
test('nothing but a test draws this export', () => {
  const files = walked()
  const wrong = refused(files).map(
    ({ at, name, by }) =>
      `${at}: ${name} is drawn by ${by.length === 0 ? 'nothing' : by.join(' and ')} and used nowhere here`,
  )
  assert.deepEqual(wrong, [])

  // A walk that read no file is a rule checked against nothing, and it passes.
  // A count cannot say which tree it walked, so one file of each module is
  // named besides, the two windows and the phone among them: a rule stops at a
  // border first.
  assert.ok(files.length > 550, `${files.length} files read: the walk is not reading the modules`)
  for (const one of [
    'modules/libs/ui/src/index.ts',
    'modules/libs/wire/index.ts',
    'modules/apps/desktop/editor/src/preset/curve.ts',
    'modules/apps/desktop/flashcards/src/decks/presets.ts',
    'modules/apps/mobile/src/core.ts',
  ]) {
    assert.ok(
      files.some(({ at }) => at === one),
      `the walk did not read ${one}, so the rule stops before it`,
    )
  }

  // Reading every file and resolving no import between them also finds
  // nothing, and passes. Two modules the rule can only know are fixtures by
  // following an import are named, one of the library and one of a window, so
  // an edge has to have been followed on either side of a package border.
  const held = fixturesOf(files)
  for (const one of [
    'modules/libs/ui/src/shared/fixtures/clock.ts',
    'modules/apps/desktop/editor/src/testing/window.ts',
  ]) {
    assert.ok(held.includes(one), `${one} is drawn by tests alone and the walk did not see it`)
  }
})

/** A barrel handing out one name, which is how a case keeps a name alive. */
const barrel = (...names) => ({
  at: 'modules/libs/ui/src/index.ts',
  text: names.map((one) => `export { ${one} } from './a'\n`).join(''),
})

/**
 * What the rule refuses, read against modules written to be refused, and what
 * it permits beside them. Every case names the whole of what the rule finds in
 * it, so nothing is permitted by accident and no case leans on another.
 */
test('what the dead-export rule refuses', () => {
  const cases = [
    {
      says: 'an export nothing but its own test draws',
      wrong: ['modules/libs/ui/src/a.ts::limiting'],
      files: [
        barrel('kept'),
        { at: 'modules/libs/ui/src/a.ts', text: 'export const limiting = () => 1\nexport const kept = () => 2\n' },
        { at: 'modules/libs/ui/src/a.test.ts', text: "import { limiting } from './a'\n" },
      ],
    },
    {
      says: 'an export nothing draws at all',
      wrong: ['modules/libs/ui/src/a.ts::placeAt'],
      files: [
        barrel('kept'),
        { at: 'modules/libs/ui/src/a.ts', text: 'export const placeAt = () => 1\nexport const kept = () => 2\n' },
      ],
    },
    {
      says: 'an export a component draws',
      wrong: [],
      files: [
        { at: 'modules/libs/ui/src/index.ts', text: "export { default as A } from './A.vue'\n" },
        { at: 'modules/libs/ui/src/a.ts', text: 'export const drawn = () => 1\n' },
        { at: 'modules/libs/ui/src/A.vue', text: "<script setup>\nimport { drawn } from './a'\n</script>" },
      ],
    },
    {
      says: 'an export the module uses where it stands',
      wrong: [],
      files: [
        barrel('kept'),
        {
          at: 'modules/libs/ui/src/a.ts',
          text: 'export interface Deps { at: number }\nexport const kept = (deps: Deps) => deps.at\n',
        },
      ],
    },
    {
      says: 'an export only a comment says the module uses',
      wrong: ['modules/libs/ui/src/a.ts::Deps'],
      files: [
        barrel('kept'),
        {
          at: 'modules/libs/ui/src/a.ts',
          text: 'export interface Deps { at: number }\n/** Handed a Deps. */\nexport const kept = () => 1\n',
        },
      ],
    },
    {
      says: 'a name the barrel hands out, drawn by nothing here',
      wrong: [],
      files: [barrel('handed'), { at: 'modules/libs/ui/src/a.ts', text: 'export const handed = () => 1\n' }],
    },
    {
      says: 'a name the barrel hands out under another name',
      wrong: [],
      files: [
        { at: 'modules/libs/ui/src/index.ts', text: "export { inner as Outer } from './a'\n" },
        { at: 'modules/libs/ui/src/a.ts', text: 'export const inner = () => 1\n' },
      ],
    },
    {
      says: 'a name the barrel hands on through a barrel of its own',
      wrong: [],
      files: [
        { at: 'modules/libs/ui/src/index.ts', text: "export * from './deep/index.ts'\n" },
        { at: 'modules/libs/ui/src/deep/index.ts', text: "export { far } from './a'\n" },
        { at: 'modules/libs/ui/src/deep/a.ts', text: 'export const far = () => 1\n' },
      ],
    },
    {
      says: 'a component the barrel hands out',
      wrong: [],
      files: [
        { at: 'modules/libs/ui/src/index.ts', text: "export { default as Plex } from './Plex.vue'\n" },
        { at: 'modules/libs/ui/src/Plex.vue', text: '<script setup>\nconst at = 1\n</script>' },
      ],
    },
    {
      says: 'a component nothing but its test draws',
      wrong: ['modules/apps/desktop/editor/src/Tab.vue::default'],
      files: [
        { at: 'modules/apps/desktop/editor/src/main.ts', text: "import App from './App.vue'\n" },
        { at: 'modules/apps/desktop/editor/src/App.vue', text: '<script setup>\nconst at = 1\n</script>' },
        { at: 'modules/apps/desktop/editor/src/Tab.vue', text: '<script setup>\nconst at = 2\n</script>' },
        { at: 'modules/apps/desktop/editor/src/Tab.test.ts', text: "import Tab from './Tab.vue'\n" },
      ],
    },
    {
      says: 'an export only a story draws',
      wrong: [],
      files: [
        barrel('kept'),
        { at: 'modules/libs/ui/src/a.ts', text: 'export const shown = () => 1\nexport const kept = () => 2\n' },
        { at: 'modules/libs/ui/src/A.stories.ts', text: "import { shown } from './a'\n" },
      ],
    },
    {
      says: 'an export of a fixture, which is written to be drawn by a test',
      wrong: [],
      files: [
        { at: 'modules/libs/ui/src/fixtures/prose.ts', text: 'export const RUSSIAN = "..."\n' },
        { at: 'modules/libs/ui/src/a.test.ts', text: "import { RUSSIAN } from './fixtures/prose'\n" },
      ],
    },
    {
      says: 'an export a window draws by the package name',
      wrong: [],
      files: [
        { at: 'modules/libs/ui/src/index.ts', text: "export { dayNamed } from './day'\n" },
        { at: 'modules/libs/ui/src/day.ts', text: 'export const dayNamed = () => 1\n' },
        { at: 'modules/apps/desktop/editor/src/main.ts', text: "import App from './App.vue'\n" },
        {
          at: 'modules/apps/desktop/editor/src/App.vue',
          text: "<script setup>\nimport { dayNamed } from '@numen/ui'\nconst at = dayNamed()\n</script>",
        },
      ],
    },
    {
      says: 'an export a module draws by the alias its own tsconfig answers for',
      wrong: [],
      files: [
        { at: 'modules/libs/ui/src/index.ts', text: "export { default as B } from './deep/B.vue'\n" },
        { at: 'modules/libs/ui/src/a.ts', text: 'export const far = () => 1\n' },
        { at: 'modules/libs/ui/src/deep/B.vue', text: "<script setup>\nimport { far } from '@/a'\n</script>" },
      ],
    },
  ]

  for (const one of cases) assert.deepEqual(found(one.files), one.wrong, one.says)
})
