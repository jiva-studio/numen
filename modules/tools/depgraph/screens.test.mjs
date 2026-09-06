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

/** The fixture window: five folders, eight edges, two of them wrong. */
const at = join(here, 'testdata/screens')

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
]

/** The fixture cruised, as `{ from → to }` for the one rule under test. */
function cruised() {
  const run = spawnSync(depcruise, ['--config', screens, '--output-type', 'json', 'src'], {
    cwd: at,
    encoding: 'utf8',
    maxBuffer: 64 * 1024 * 1024,
  })
  assert.ok(run.stdout, `the fixture cruise did not run: ${run.stderr?.trim() || run.error}`)
  const out = JSON.parse(run.stdout)
  return {
    refused: out.summary.violations
      .filter((one) => one.rule.name === 'no-screen-reaches-a-screen')
      .map((one) => `${one.from} → ${one.to}`),
    read: out.modules.map((one) => one.source),
  }
}

test('what the screen rule refuses, and what it lets through', () => {
  const { refused, read } = cruised()

  // A cruise that read nothing refuses nothing and says so in the same words as
  // a clean one. Every file of the fixture is named, so a walk that lost one of
  // them fails here rather than passing quietly.
  for (const file of [
    'src/words.ts',
    'src/notices/telling.ts',
    'src/tabs/putting.ts',
    'src/note/tab.ts',
    'src/note/inner/deep.ts',
    'src/cards/deck.ts',
    'src/ledger/entries.ts',
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
})
