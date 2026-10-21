/**
 * Every item either menu offers is drawn with an icon.
 *
 * A menu keeps the room for one on every row, so an item without one is a gap
 * where its neighbours have a picture.
 */
import { describe, expect, it } from 'vitest'

import { findIcon } from '@/shared/icons'
import { getNoteIcon } from '@/entities/note'
import { createCommands, createPathCommands } from '@/features/command-palette'
import { createMenuItems } from '@plugins/views.file.tree/frontend'
import { ITEMS, NONE } from '@plugins/views.plex.graph/frontend'
import { activate, PRESET } from '@plugins/views.flashcards.studio/frontend'
import {
  createMockBufferHost,
  createMockPluginContext,
  type WorkspaceHost,
} from '@numen/plugin-sdk'
import { createWelcomeActions } from '@/pages/welcome'
import { WORDS as words } from '@/shared/words'
import type { NoteType, Source } from '@/entities/file'

/** A window that has been told nothing, which can do every run. */
const canRunAnything = () => true

/** The menu the tree draws on a row: what the tab does, among the window's commands. */
const getTreeMenu = (source: Source, isFolder: boolean, isSeveral = false) =>
  createMenuItems(createPathCommands({ source, isFolder }, isSeveral, canRunAnything), isSeveral)

/** Every menu the tree draws: off every row, and on a row of each kind. */
const inTheTree = [
  createMenuItems(null, false),
  ...(['note', 'book', 'recording', 'url', 'other'] as Source[]).flatMap((source) => [
    getTreeMenu(source, false),
    getTreeMenu(source, true),
  ]),
  getTreeMenu('note', false, true),
].flat()

describe('the icon a command is drawn with', () => {
  it('is there for every item the tree offers', () => {
    for (const item of inTheTree) expect(findIcon(item.id), item.id).not.toBeNull()
  })

  it('is there for every item the plex offers, on a node and off every node', () => {
    for (const item of [...ITEMS, ...NONE]) expect(findIcon(item.id), item.id).not.toBeNull()
  })

  it('is the same one wherever a command is offered', () => {
    const onANode = new Map(ITEMS.map((item) => [item.id, findIcon(item.id)]))
    for (const item of inTheTree) {
      const drawn = onANode.get(item.id)
      if (drawn) expect(findIcon(item.id), item.id).toBe(drawn)
    }
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

// One mark to a kind: a preset is the same thing in the tree, in the plex and
// in the tab it opens in.
describe('the icon of a kind of note', () => {
  const kinds: readonly NoteType[] = ['note', 'deck', 'stencil', 'preset']

  it('is there for every kind a note may be', () => {
    for (const kind of kinds) expect(getNoteIcon(kind), kind).toBeTruthy()
  })

  it('tells the five kinds apart', () => {
    expect(new Set(kinds.map(getNoteIcon)).size).toBe(kinds.length)
  })

  it('draws a preset as the tab that opens it does', () => {
    const context = createMockPluginContext({
      workspace: {
        getMessageWriter: () => () => undefined,
        getToday: () => '2026-09-04',
      } as unknown as WorkspaceHost,
      buffers: createMockBufferHost(),
    })
    activate(context)

    const kind = context.getRegisteredTabKinds().find((one) => one.kind === PRESET)

    expect(getNoteIcon('preset')).toBe(kind?.icon)
  })
})
