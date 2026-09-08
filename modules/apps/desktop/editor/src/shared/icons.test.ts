/**
 * Every item either menu offers is drawn with an icon.
 *
 * A menu keeps the room for one on every row, so an item without one is a gap
 * where its neighbours have a picture.
 */
import { describe, expect, it } from 'vitest'

import { iconFor, iconOfKind, iconOfNote } from './icons'
import { PRESET } from './tabs/workspace'
import { commandsOf } from './command/commands'
import { itemsFor } from '../files/menu'
import { ITEMS, NONE } from '../plex/menu'
import { waysIn } from '../welcome/screen'
import { WORDS as words } from './words'
import type { NoteType, Source } from './core'

/** A window that has been told nothing, which can do every run. */
const anything = () => true

/** Every menu the tree draws: off every row, and on a row of each kind. */
const inTheTree = [
  itemsFor(null, false, anything),
  ...(['note', 'book', 'recording', 'other'] as Source[]).flatMap((source) => [
    itemsFor({ source, folder: false }, false, anything),
    itemsFor({ source, folder: true }, false, anything),
  ]),
  itemsFor({ source: 'note', folder: false }, true, anything),
].flat()

describe('the icon a command is drawn with', () => {
  it('is there for every item the tree offers', () => {
    for (const item of inTheTree) expect(iconFor(item.id), item.id).not.toBeNull()
  })

  it('is there for every item the plex offers, on a node and off every node', () => {
    for (const item of [...ITEMS, ...NONE]) expect(iconFor(item.id), item.id).not.toBeNull()
  })

  it('is the same one wherever a command is offered', () => {
    const onANode = new Map(ITEMS.map((item) => [item.id, iconFor(item.id)]))
    for (const item of inTheTree) {
      const drawn = onANode.get(item.id)
      if (drawn) expect(iconFor(item.id), item.id).toBe(drawn)
    }
  })

  it('is there for every command the palette draws', () => {
    for (const one of commandsOf(words)) expect(iconFor(one.id), one.id).not.toBeNull()
  })

  it('is there for every way into a vault the welcome screen offers', () => {
    const ways = waysIn({ vault: 'physics', ready: true }, words, 'Linux')
    for (const one of ways) expect(iconFor(one.id), one.id).not.toBeNull()
  })

  it('is nothing for an identity no list carries', () => {
    expect(iconFor('constructor')).toBeNull()
    expect(iconFor('nothing of the sort')).toBeNull()
  })
})

// One mark to a kind: a preset is the same thing in the tree, in the plex and
// in the tab it opens in.
describe('the icon of a kind of note', () => {
  const kinds: readonly NoteType[] = ['note', 'deck', 'stencil', 'preset']

  it('is there for every kind a note may be', () => {
    for (const kind of kinds) expect(iconOfNote(kind), kind).toBeTruthy()
  })

  it('tells the five kinds apart', () => {
    expect(new Set(kinds.map(iconOfNote)).size).toBe(kinds.length)
  })

  it('draws a preset as the tab that opens it does', () => {
    expect(iconOfNote('preset')).toBe(iconOfKind(PRESET))
  })
})
