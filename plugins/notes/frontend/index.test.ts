import { describe, expect, it, vi } from 'vitest'
import { NOTE_EDITOR, type HostServices } from '@numen/plugin-sdk'
import { activate, deactivate } from './index'
import { createEditorSettingsContext } from './testing/editorSettings'

/** Host services whose vault offers no notes. */
const NO_NOTES = { vault: {} } as unknown as HostServices

describe('notes plugin lifecycle', () => {
  it('registers markdown reader and note tab kind on activation', () => {
    const { context } = createEditorSettingsContext({ lineNumbers: false, softWrap: true })

    activate(context, NO_NOTES)

    expect(context.register.reader).toHaveBeenCalledWith(
      expect.objectContaining({
        id: 'markdown',
        title: 'Markdown Note Editor',
      }),
    )

    expect(context.register.tabKind).toHaveBeenCalledWith(
      expect.objectContaining({
        kind: 'note',
        title: 'Note',
      }),
    )
  })

  it('registers markdown reader and note tab kind before the editor settings have arrived', () => {
    const { context } = createEditorSettingsContext(undefined)

    activate(context, NO_NOTES)

    expect(context.register.reader).toHaveBeenCalledOnce()
    expect(context.register.tabKind).toHaveBeenCalledOnce()
  })

  it('registers no command, since creating a note is a command of the editor', () => {
    const { context } = createEditorSettingsContext({ lineNumbers: false, softWrap: true })

    activate(context, NO_NOTES)

    expect(context.register.command).not.toHaveBeenCalled()
  })

  it('registers the note-editor view where the host offers the notes of the vault', () => {
    const { context } = createEditorSettingsContext({ lineNumbers: false, softWrap: true })
    const host = { vault: { notes: { read: vi.fn(), write: vi.fn() } } } as unknown as HostServices

    activate(context, host)

    expect(context.register.view).toHaveBeenCalledWith(
      expect.objectContaining({ capability: NOTE_EDITOR }),
    )
  })

  it('runs deactivation hook cleanly', () => {
    expect(() => deactivate()).not.toThrow()
  })
})
