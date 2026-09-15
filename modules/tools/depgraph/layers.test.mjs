/**
 * What the layer rules refuse, and what they have to let through.
 *
 * A boundary rule is only worth what it permits: one that refuses a slice
 * reaching a slice and also refuses a slice reaching the shared layer would be
 * met by moving every folder back into the root. So each fixture holds one edge
 * of each kind the tree actually has, and the runs below name every one of them.
 *
 * Three layouts answer to one config. `layers` is a tree laid out in layers,
 * `screens` a window laid out flat, where a folder naming no layer is a screen,
 * and `ring` two folders that each reach the other.
 */
import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { join } from 'node:path'
import test from 'node:test'
import { depcruise, here, layers } from './modules.mjs'

/** One fixture cruised: what each rule refused, and the files the walk read. */
function cruised(name) {
  const run = spawnSync(depcruise, ['--config', layers, '--output-type', 'json', 'src'], {
    cwd: join(here, 'testdata', name),
    encoding: 'utf8',
    maxBuffer: 64 * 1024 * 1024,
  })
  assert.ok(run.stdout, `the ${name} cruise did not run: ${run.stderr?.trim() || run.error}`)
  const out = JSON.parse(run.stdout)
  return {
    refused: out.summary.violations.map((one) => ({
      by: one.rule.name,
      edge: `${one.from} → ${one.to}`,
    })),
    read: out.modules.map((one) => one.source),
  }
}

/**
 * Whether a rule reads which way an edge points, which every rule does but the
 * ring. The names cannot be read off the config from here: it builds its door
 * rules from the folders of whatever tree it is loaded in, and a fixture's
 * folders are not this one's.
 */
const direction = (one) => one !== 'no-folder-going-round'

/** Every file a fixture holds is named, so a walk that lost one fails here. */
function readAll(read, files, name) {
  for (const file of files) {
    assert.ok(read.includes(file), `the ${name} cruise did not read ${file}`)
  }
}

/**
 * Every edge a fixture holds is accounted for, so an edge that stopped being
 * drawn cannot pass as an edge that stopped being refused.
 *
 * `only` names the rules asked about. A folder ring is the consequence of the
 * upward edges a fixture holds on purpose, and reading it here would say the
 * direction rules found something they did not.
 */
function accountedFor(refused, edges, only) {
  const held = only ? refused.filter((one) => only(one.by)) : refused
  for (const one of edges) {
    const found = held.find((wrong) => wrong.edge === one.edge)
    if (one.by === null) {
      assert.equal(found, undefined, `${one.says} was refused: ${one.edge}`)
      continue
    }
    assert.ok(found, `${one.says} was let through: ${one.edge}`)
    assert.equal(found.by, one.by, `${one.says} was refused by ${found.by}`)
  }
  assert.deepEqual(
    held.map((one) => one.edge).sort(),
    edges.filter((one) => one.by !== null).map((one) => one.edge).sort(),
  )
}

/** The layered fixture: three layers, ten edges, four of them wrong. */
const layered = [
  {
    says: 'a feature reaching another feature',
    by: 'no-features-slice-reaches-a-slice',
    edge: 'src/features/cards/deck.ts → src/features/thread/turn.ts',
  },
  {
    says: 'a shared module reaching a feature',
    by: 'no-shared-reaches-above-itself',
    edge: 'src/shared/lib/sorting.ts → src/features/cards/index.ts',
  },
  {
    says: 'a shared module reaching a screen',
    by: 'no-shared-reaches-above-itself',
    edge: 'src/shared/lib/sorting.ts → src/screens/agent.ts',
  },
  {
    says: 'a feature reaching a screen',
    by: 'no-features-reaches-above-itself',
    edge: 'src/features/cards/counting.ts → src/screens/agent.ts',
  },
  {
    says: 'a feature reaching the shared layer',
    by: null,
    edge: 'src/features/cards/deck.ts → src/shared/lib/place.ts',
  },
  {
    says: "a folder under a feature reaching that feature's own top",
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
    says: 'a screen reaching a feature through its door',
    by: null,
    edge: 'src/screens/agent.ts → src/features/thread/index.ts',
  },
  {
    says: 'a screen reaching the shared layer',
    by: null,
    edge: 'src/screens/agent.ts → src/shared/ui/menu/item.ts',
  },
  {
    says: 'the barrel at the root reaching a feature through its door',
    by: null,
    edge: 'src/index.ts → src/features/cards/index.ts',
  },
  {
    says: 'a screen reaching past a feature\'s door',
    by: 'no-reaching-past-a-features-slices-door',
    edge: 'src/screens/agent.ts → src/features/cards/deck-editor/editor.ts',
  },
]

test('a tree laid out in layers', () => {
  const { refused, read } = cruised('layers')
  readAll(
    read,
    [
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
    ],
    'layers',
  )
  accountedFor(refused, layered, direction)
})

/** The flat fixture: a folder naming no layer is a screen. */
const flat = [
  {
    says: 'a screen reaching another screen',
    by: 'no-screen-reaches-a-screen',
    edge: 'src/cards-tab/deck.ts → src/note-tab/tab.ts',
  },
  {
    says: 'a folder the rule holds no part reaching a screen',
    by: 'no-screen-reaches-a-screen',
    edge: 'src/ledger/entries.ts → src/note-tab/tab.ts',
  },
  {
    says: 'a screen reaching the top of shared',
    by: null,
    edge: 'src/cards-tab/deck.ts → src/shared/core.ts',
  },
  {
    says: 'a screen reaching a folder under shared',
    by: null,
    edge: 'src/note-tab/tab.ts → src/shared/tabs/putting.ts',
  },
  {
    says: "a screen reaching the window's root",
    by: null,
    edge: 'src/note-tab/tab.ts → src/words.ts',
  },
  {
    says: "a folder under a screen reaching that screen's own top",
    by: null,
    edge: 'src/note-tab/inner/deep.ts → src/note-tab/tab.ts',
  },
  {
    says: 'one folder under shared reaching another',
    by: null,
    edge: 'src/shared/tabs/putting.ts → src/shared/notices/telling.ts',
  },
  {
    says: "a folder under shared reaching the window's root",
    by: null,
    edge: 'src/shared/notices/telling.ts → src/words.ts',
  },
  {
    says: 'the shell mounting a screen',
    by: null,
    edge: 'src/window/mounting.ts → src/note-tab/tab.ts',
  },
  {
    says: 'the shell mounting a second screen',
    by: null,
    edge: 'src/window/mounting.ts → src/cards-tab/deck.ts',
  },
]

test('a window laid out flat, where every folder is a screen', () => {
  const { refused, read } = cruised('screens')
  readAll(
    read,
    [
      'src/words.ts',
      'src/shared/core.ts',
      'src/shared/notices/telling.ts',
      'src/shared/tabs/putting.ts',
      'src/window/mounting.ts',
      'src/note-tab/tab.ts',
      'src/note-tab/inner/deep.ts',
      'src/cards-tab/deck.ts',
      'src/ledger/entries.ts',
    ],
    'screens',
  )
  accountedFor(refused, flat, direction)
})

/**
 * A ring that runs through the folder boundary and through no file. Both
 * folders are under `shared/`, so no direction rule has anything to say here,
 * and a ring caught by the wrong rule would say the folder check works when it
 * does not.
 */
test('two folders that each reach the other', () => {
  const { refused, read } = cruised('ring')
  readAll(
    read,
    ['src/shared/tabs/putting.ts', 'src/shared/tabs/marking.ts', 'src/shared/notices/telling.ts'],
    'ring',
  )
  accountedFor(refused, [
    {
      says: 'one folder under shared reaching the folder that reaches it',
      by: 'no-folder-going-round',
      edge: 'src/shared/notices → src/shared/tabs',
    },
    {
      says: 'the other way round the same ring',
      by: 'no-folder-going-round',
      edge: 'src/shared/tabs → src/shared/notices',
    },
  ])
})
