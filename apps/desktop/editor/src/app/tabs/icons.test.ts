/**
 * Every item either menu offers is drawn with an icon.
 *
 * A menu keeps the room for one on every row, so an item without one is a gap
 * where its neighbours have a picture.
 */
import { describe, expect, it } from 'vitest'

import { findIcon } from '@/shared/icons'
import { getNoteIcon, registerNoteIcon } from '@/entities/note'
import { createCommands, createPathCommands } from '@/features/command-palette'
import { createWelcomeActions } from '@/pages/welcome'
import { WORDS as words } from '@/shared/words'

/** An item of a menu, and the items it opens. */
type Item = { readonly id: string; readonly children?: readonly Item[] | undefined }

const flatten = (items: readonly Item[]): readonly Item[] =>
  items.flatMap((item) => [item, ...flatten(item.children ?? [])])

/** The window's commands over a row: on a row of each kind, and over several rows. */
const onARow = flatten(
  [
    ...[true, false].flatMap((isNote) => [
      createPathCommands({ isNote, isFolder: false }, false),
      createPathCommands({ isNote, isFolder: true }, false),
    ]),
    createPathCommands({ isNote: true, isFolder: false }, true),
  ].flat(),
)

describe('the icon a command is drawn with', () => {
  it('is there for every command the window offers over a row', () => {
    for (const item of onARow) expect(findIcon(item.id), item.id).not.toBeNull()
  })

  it('is there for every command the palette draws', () => {
    for (const one of createCommands(words)) expect(findIcon(one.id), one.id).not.toBeNull()
  })

  it('is there for every way into a vault the welcome screen offers', () => {
    const ways = createWelcomeActions({ vault: 'physics', isReady: true }, words, 'Linux')
    for (const one of ways) expect(findIcon(one.id), one.id).not.toBeNull()
  })

  it('is nothing for an identity no list carries', () => {
    expect(findIcon('constructor')).toBeNull()
    expect(findIcon('nothing of the sort')).toBeNull()
  })
})

// One mark to a type.
describe('the icon of a type of note', () => {
  it('is the icon of a note for a type no plugin draws', () => {
    expect(getNoteIcon('sample')).toBe(getNoteIcon('note'))
  })

  it('is the icon a plugin registers for its type, until the plugin lets go of it', () => {
    const icon = {}
    const unregister = registerNoteIcon('sample', icon)
    expect(getNoteIcon('sample')).toBe(icon)

    unregister()
    expect(getNoteIcon('sample')).toBe(getNoteIcon('note'))
  })
})
