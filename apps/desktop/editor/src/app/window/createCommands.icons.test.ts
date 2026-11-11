/**
 * Every command the palette and the welcome screen draw has an icon.
 *
 * A menu reserves the icon column on every row, so an item without an icon
 * leaves a gap where its neighbors have one.
 */
import { describe, expect, it } from 'vitest'

import { COMMAND, findIcon } from '@/shared/model/command'
import { getNamedIcon } from '@/shared/ui/icons'
import { getNoteIcon, registerTabKindIcon } from '@/entities/note'
import { createAppCommands } from '@/testing/appCommands'

describe('the icon a command is drawn with', () => {
  it('is there for every command the window places in the menu of a row of the file tree', () => {
    for (const id of [COMMAND.openFile, COMMAND.copyFilePath, COMMAND.removeFile]) {
      expect(findIcon(id), id).not.toBeNull()
    }
  })

  it('is there for every command the palette draws', () => {
    for (const one of createAppCommands()) expect(findIcon(one.id), one.id).not.toBeNull()
  })

  it('is there for declared plugin commands and plugin settings, with distinct icons', () => {
    expect(findIcon('org.example.custom:task.run')).not.toBeNull()
    expect(findIcon('setting:org.example.custom:toggle')).not.toBeNull()
    expect(findIcon(COMMAND.openSettings)).not.toBe(findIcon('setting:org.example.custom:toggle'))
    expect(getNamedIcon('layers')).not.toBe(getNamedIcon('layout-template'))
    expect(getNamedIcon('audio-lines')).not.toBe(getNamedIcon('waypoints'))
  })

  it('is nothing for a command of the window that no icon is kept for', () => {
    expect(findIcon(COMMAND.moveFile)).toBeNull()
    expect(findIcon('mode:dark')).toBeNull()
  })

  it('is nothing for an identity no list carries', () => {
    expect(findIcon('constructor')).toBeNull()
    expect(findIcon('nothing of the sort')).toBeNull()
  })
})

describe('the icon of a note', () => {
  it('is the icon of a note for a tab kind no plugin draws, and for none', () => {
    expect(getNoteIcon('sample')).toBe(getNoteIcon(undefined))
  })

  it('is the icon a plugin registers for the tab kind that opens the note, until it unregisters it', () => {
    const icon = {}
    const unregister = registerTabKindIcon('sample', icon)
    expect(getNoteIcon('sample')).toBe(icon)

    unregister()
    expect(getNoteIcon('sample')).toBe(getNoteIcon(undefined))
  })
})
