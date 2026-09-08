/**
 * What the screen rule refuses, and what it has to let through.
 *
 * A boundary rule is only worth what it permits: one that refuses a screen
 * reaching a screen and also refuses a screen reaching the tabs would be met by
 * moving every folder back into the root. So the fixture under `testdata/`
 * holds one edge of each kind the windows actually have, and the run below
 * names every one of them.
 */
import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { join } from 'node:path'
import test from 'node:test'
import { depcruise, here, screens } from './modules.mjs'

/** The fixture window, which is filed both ways a window of ours is filed. */
const at = join(here, 'testdata/screens')

/** Two folders that each reach the other, where no one file is in a cycle. */
const ring = join(here, 'testdata/ring')

/** Every edge the fixture holds, and whether the rule is meant to refuse it. */
const edges = [
  { says: 'a screen reaching another screen', refused: true, edge: 'src/cards/deck.ts → src/note/tab.ts' },
  {
    says: 'a folder nobody named as shared reaching a screen',
    refused: true,
    edge: 'src/ledger/entries.ts → src/note/tab.ts',
  },
  { says: 'a screen reaching a shared folder', refused: false, edge: 'src/cards/deck.ts → src/tabs/putting.ts' },
  { says: 'a screen reaching the window\'s root', refused: false, edge: 'src/note/tab.ts → src/words.ts' },
  {
    says: 'a folder under a screen reaching that screen\'s own top',
    refused: false,
    edge: 'src/note/inner/deep.ts → src/note/tab.ts',
  },
  {
    says: 'a shared folder reaching another shared folder',
    refused: false,
    edge: 'src/tabs/putting.ts → src/notices/telling.ts',
  },
  {
    says: 'a shared folder reaching the window\'s root',
    refused: false,
    edge: 'src/notices/telling.ts → src/words.ts',
  },
  {
    says: 'a screen under features reaching another screen under features',
    refused: true,
    edge: 'src/features/media/kind.ts → src/features/plex/view.ts',
  },
  {
    says: 'a screen under features reaching a screen the window kept at its root',
    refused: true,
    edge: 'src/features/media/kind.ts → src/note/tab.ts',
  },
  {
    says: 'a screen under features reaching the shared folder',
    refused: false,
    edge: 'src/features/media/kind.ts → src/shared/core.ts',
  },
  {
    says: 'a screen under features reaching a folder of its own',
    refused: false,
    edge: 'src/features/media/kind.ts → src/features/media/url-tab/frame.ts',
  },
  {
    says: 'a folder under a screen under features reaching that screen\'s own top',
    refused: false,
    edge: 'src/features/media/url-tab/frame.ts → src/features/media/words.ts',
  },
  {
    says: 'a second screen under features reaching the shared folder',
    refused: false,
    edge: 'src/features/plex/view.ts → src/shared/core.ts',
  },
]

/** One fixture cruised, with the edges each rule refused and the files read. */
function cruised(where) {
  const run = spawnSync(depcruise, ['--config', screens, '--output-type', 'json', 'src'], {
    cwd: where,
    encoding: 'utf8',
    maxBuffer: 64 * 1024 * 1024,
  })
  assert.ok(run.stdout, `the fixture cruise did not run: ${run.stderr?.trim() || run.error}`)
  const out = JSON.parse(run.stdout)
  const under = (rule) =>
    out.summary.violations.filter((one) => one.rule.name === rule).map((one) => `${one.from} → ${one.to}`)
  return {
    refused: under('no-screen-reaches-a-screen'),
    rings: under('no-folder-going-round'),
    read: out.modules.map((one) => one.source),
  }
}

test('what the screen rule refuses, and what it lets through', () => {
  const { refused, rings, read } = cruised(at)

  // A cruise that read nothing refuses nothing and says so in the same words as
  // a clean one. Every file of the fixture is named, so a walk that lost one of
  // them fails here.
  for (const file of [
    'src/words.ts',
    'src/notices/telling.ts',
    'src/tabs/putting.ts',
    'src/note/tab.ts',
    'src/note/inner/deep.ts',
    'src/cards/deck.ts',
    'src/ledger/entries.ts',
    'src/shared/core.ts',
    'src/features/plex/view.ts',
    'src/features/media/kind.ts',
    'src/features/media/words.ts',
    'src/features/media/url-tab/frame.ts',
  ]) {
    assert.ok(read.includes(file), `the fixture cruise did not read ${file}`)
  }

  // Every edge the fixture holds is accounted for, so an edge that stopped
  // being drawn cannot pass as an edge that stopped being refused.
  for (const one of edges) {
    assert.equal(
      refused.includes(one.edge),
      one.refused,
      one.refused ? `${one.says} was let through: ${one.edge}` : `${one.says} was refused: ${one.edge}`,
    )
  }

  assert.deepEqual(
    refused.sort(),
    edges.filter((one) => one.refused).map((one) => one.edge).sort(),
  )

  // The two rules are separate: nothing in this fixture is a ring.
  assert.deepEqual(rings, [])
})

/**
 * A ring that runs through the folder boundary and through no file. The
 * file-level rule cannot see it: neither `putting.ts` nor `telling.ts` is in a
 * cycle, and the folder-level one is what refuses it.
 */
test('two folders that each reach the other', () => {
  const { refused, rings, read } = cruised(ring)

  for (const file of ['src/tabs/putting.ts', 'src/tabs/marking.ts', 'src/notices/telling.ts']) {
    assert.ok(read.includes(file), `the ring fixture cruise did not read ${file}`)
  }

  assert.deepEqual(rings.sort(), ['src/notices → src/tabs', 'src/tabs → src/notices'])

  // Both folders are shared, so the screen rule has nothing to say here. A
  // ring caught by the wrong rule would say the folder check works when it
  // does not.
  assert.deepEqual(refused, [])
})
