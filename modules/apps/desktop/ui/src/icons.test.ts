/**
 * Every item either menu offers is drawn with an icon.
 *
 * A menu keeps the room for one on every row, so an item without one is a gap
 * where its neighbours have a picture.
 */
import { describe, expect, it } from 'vitest'

import { iconFor } from './icons'
import { itemsFor } from './files/menu'
import { ITEMS, NONE } from './plex/menu'
import type { Source } from './core'

/** Every menu the tree draws: off every row, and on a row of each kind. */
const inTheTree = [
  itemsFor(null, false),
  ...(['note', 'book', 'other'] as Source[]).flatMap((source) => [
    itemsFor({ source, folder: false }, false),
    itemsFor({ source, folder: true }, false),
  ]),
  itemsFor({ source: 'note', folder: false }, true),
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

  it('is nothing for a command no menu offers', () => {
    expect(iconFor('destroy')).toBeNull()
    expect(iconFor('constructor')).toBeNull()
  })
})
