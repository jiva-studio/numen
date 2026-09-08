/**
 * What the layer rules refuse, and what they have to let through.
 *
 * A boundary rule is only worth what it permits: one that refuses a feature
 * reaching a feature and also refuses a feature reaching the shared layer would
 * be met by moving every folder back into the root. So the fixture under
 * `testdata/layers` holds one edge of each kind the library actually has, and
 * the run below names every one of them.
 */
import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { join } from 'node:path'
import test from 'node:test'
import { depcruise, here, layers } from './modules.mjs'

/** The fixture library: three layers, ten edges, four of them wrong. */
const at = join(here, 'testdata/layers')

/** Every edge the fixture holds, and which rule is meant to refuse it. */
const edges = [
  {
    says: 'a feature reaching another feature',
    by: 'no-feature-reaches-a-feature',
    edge: 'src/features/cards/deck.ts → src/features/thread/turn.ts',
  },
  {
    says: 'a shared module reaching a feature',
    by: 'no-shared-reaches-above-itself',
    edge: 'src/shared/lib/sorting.ts → src/features/cards/deck.ts',
  },
  {
    says: 'a shared module reaching a screen',
    by: 'no-shared-reaches-above-itself',
    edge: 'src/shared/lib/sorting.ts → src/screens/agent.ts',
  },
  {
    says: 'a feature reaching a screen',
    by: 'no-feature-reaches-a-screen',
    edge: 'src/features/cards/counting.ts → src/screens/agent.ts',
  },
  {
    says: 'a feature reaching the shared layer',
    by: null,
    edge: 'src/features/cards/deck.ts → src/shared/lib/place.ts',
  },
  {
    says: 'a folder under a feature reaching that feature\'s own top',
    by: null,
    edge: 'src/features/cards/deck-editor/editor.ts → src/features/cards/deck.ts',
  },
  {
    says: 'one shared component drawing another',
    by: null,
    edge: 'src/shared/ui/select/select.ts → src/shared/ui/menu/item.ts',
  },
  {
    says: 'a shared component reaching the shared library',
    by: null,
    edge: 'src/shared/ui/menu/item.ts → src/shared/lib/place.ts',
  },
  {
    says: 'a screen reaching a feature',
    by: null,
    edge: 'src/screens/agent.ts → src/features/thread/turn.ts',
  },
  {
    says: 'a screen reaching the shared layer',
    by: null,
    edge: 'src/screens/agent.ts → src/shared/ui/menu/item.ts',
  },
  {
    says: 'the barrel at the root reaching a feature',
    by: null,
    edge: 'src/index.ts → src/features/cards/deck.ts',
  },
]

/** Every file the fixture holds, so a walk that lost one of them fails here. */
const files = [
  'src/index.ts',
  'src/shared/lib/place.ts',
  'src/shared/lib/sorting.ts',
  'src/shared/ui/menu/item.ts',
  'src/shared/ui/select/select.ts',
  'src/features/thread/turn.ts',
  'src/features/cards/deck.ts',
  'src/features/cards/counting.ts',
  'src/features/cards/deck-editor/editor.ts',
  'src/screens/agent.ts',
]

const run = spawnSync(depcruise, ['--config', layers, '--output-type', 'json', 'src'], {
  cwd: at,
  encoding: 'utf8',
  maxBuffer: 64 * 1024 * 1024,
})
assert.ok(run.stdout, `the fixture cruise did not run: ${run.stderr?.trim() || run.error}`)
const out = JSON.parse(run.stdout)

/** Each violation as the rule that refused it and the edge it refused. */
const refused = out.summary.violations.map((one) => ({
  by: one.rule.name,
  edge: `${one.from} → ${one.to}`,
}))

test('what the layer rules refuse, and what they let through', () => {
  // A cruise that read nothing refuses nothing and says so in the same words as
  // a clean one.
  const read = out.modules.map((one) => one.source)
  for (const file of files) {
    assert.ok(read.includes(file), `the fixture cruise did not read ${file}`)
  }

  // Every edge the fixture holds is accounted for, so an edge that stopped
  // being drawn cannot pass as an edge that stopped being refused.
  for (const one of edges) {
    const found = refused.find((wrong) => wrong.edge === one.edge)
    if (one.by === null) {
      assert.equal(found, undefined, `${one.says} was refused: ${one.edge}`)
      continue
    }
    assert.ok(found, `${one.says} was let through: ${one.edge}`)
    assert.equal(found.by, one.by, `${one.says} was refused by ${found.by}`)
  }
})

test('the fixture holds nothing the rules were not told about', () => {
  assert.deepEqual(
    refused.map((one) => one.edge).sort(),
    edges.filter((one) => one.by !== null).map((one) => one.edge).sort(),
  )
})
