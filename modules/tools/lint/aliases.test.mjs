import assert from 'node:assert/strict'
import test from 'node:test'
import { owed, renames } from './aliases.mjs'
import { sources } from './source.mjs'

/**
 * The vocabulary is settled at the border, not at the call site. A window that
 * has to rename what it takes from the library has met a name the two of them
 * both want, and the duplicate-name checks never saw it: both declarations are
 * spelled correctly, each in its own module.
 */
test('no name is renamed on its way across our own border', () => {
  const wrong = []
  const read = []
  for (const { at, text } of sources(['.ts', '.vue'])) {
    read.push(at)
    for (const one of renames(text)) {
      const said = `${at}: ${one}`
      if (!owed.includes(said)) wrong.push(said)
    }
  }
  assert.deepEqual(wrong, [])

  // A walk that read no file of the modules is a rule checked against nothing,
  // and it passes. A count alone cannot say which modules it read, so the
  // furthest of them is named: the phone takes from the library too.
  assert.ok(
    read.length > 300,
    `${read.length} files of the interface modules read: the walk is not reading them`,
  )
  assert.ok(
    read.some((at) => at.endsWith('apps/mobile/src/plex/picture.ts')),
    "the walk did not read the phone's picture.ts, so the rule stops at the mobile border",
  )

  // An entry naming a border nothing renames across any more is a rule kept
  // alive by a line nobody reads. The list only shrinks.
  const standing = new Set()
  for (const { at, text } of sources(['.ts', '.vue'])) {
    for (const one of renames(text)) standing.add(`${at}: ${one}`)
  }
  assert.deepEqual(owed.filter((one) => !standing.has(one)), [])
})

/**
 * What the rule refuses, read against clauses written to be refused. The names
 * it lets through are the ones we did not choose and the ones that never
 * crossed a border.
 */
test('what the alias rule refuses', () => {
  const cases = [
    {
      says: 'a component of ours renamed by the window that mounts it',
      allowed: false,
      source: "import { Palette as PaletteView } from '@numen/ui'",
    },
    {
      says: 'a type of ours renamed on the way in',
      allowed: false,
      source: "import type { Row as TreeRow } from '@numen/ui'",
    },
    {
      says: 'a name of ours re-exported under another',
      allowed: false,
      source: "export { stepTo as stepToRow } from '@numen/ui'",
    },
    {
      says: 'a name of ours taken as it is written',
      allowed: true,
      source: "import { Palette, Menu } from '@numen/ui'",
    },
    {
      says: "a third-party package's name disambiguated",
      allowed: true,
      source: "import { tags as t } from '@lezer/highlight'",
    },
    {
      says: 'a name of the generated schema, which is not ours to choose',
      allowed: true,
      source: "import { Vault as VaultMessage } from '@numen/protocol'",
    },
    {
      says: 'a neighbouring file of the same package',
      allowed: true,
      source: "import { Sheet as Paper } from './strip'",
    },
    {
      says: 'the default export, which carries no name to keep',
      allowed: true,
      source: "import { default as Plex } from '@numen/ui'",
    },
    {
      says: 'a whole package taken under one name',
      allowed: true,
      source: "import * as ui from '@numen/ui'",
    },
  ]

  const refused = cases.filter((one) => renames(one.source).length > 0).map((one) => one.says)
  const wanted = cases.filter((one) => !one.allowed).map((one) => one.says)
  assert.deepEqual(refused, wanted)
})
