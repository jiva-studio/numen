import { describe, expect, it, vi } from 'vitest'
import { NOTE_EDITOR, type PluginContext } from '@numen/plugin-sdk'
import { activate, deactivate } from './index'

describe('notes plugin lifecycle', () => {
  it('registers markdown reader and note tab kind on activation', () => {
    const registerReader = vi.fn()
    const registerTabKind = vi.fn()
    const context = {
      registerReader,
      registerTabKind,
    } as unknown as PluginContext

    activate(context)

    expect(registerReader).toHaveBeenCalledWith(
      expect.objectContaining({
        id: 'markdown',
        title: 'Markdown Note Editor',
      }),
    )

    expect(registerTabKind).toHaveBeenCalledWith(
      expect.objectContaining({
        kind: 'note',
        title: 'Note',
      }),
    )
  })

  it('registers the note-editor view where the host offers the notes of the vault', () => {
    const registerView = vi.fn()
    const context = {
      registerReader: vi.fn(),
      registerTabKind: vi.fn(),
      registerView,
      notes: { read: vi.fn(), write: vi.fn() },
    } as unknown as PluginContext

    activate(context)

    expect(registerView).toHaveBeenCalledWith(expect.objectContaining({ capability: NOTE_EDITOR }))
  })

  it('runs deactivation hook cleanly', () => {
    expect(() => deactivate()).not.toThrow()
  })
})
