import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { EditorView } from '@codemirror/view'
import { NOTE_EDITOR, type PluginContext } from '@numen/plugin-sdk'
import { activate, deactivate, MarkdownNoteEditor, readMarkdownEditorPreferences } from './index'

describe('notes plugin lifecycle', () => {
  it('registers markdown reader and note tab kind on activation', () => {
    const registerReader = vi.fn()
    const registerTabKind = vi.fn()
    const context = {
      register: { reader: registerReader, tabKind: registerTabKind },
      vault: {},
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

  it('registers no command, since creating a note is a command of the editor', () => {
    const registerCommand = vi.fn()
    const context = {
      register: { reader: vi.fn(), tabKind: vi.fn(), command: registerCommand },
      vault: {},
    } as unknown as PluginContext

    activate(context)

    expect(registerCommand).not.toHaveBeenCalled()
  })

  it('registers the note-editor view where the host offers the notes of the vault', () => {
    const registerView = vi.fn()
    const context = {
      register: { reader: vi.fn(), tabKind: vi.fn(), view: registerView },
      vault: { notes: { read: vi.fn(), write: vi.fn() } },
    } as unknown as PluginContext

    activate(context)

    expect(registerView).toHaveBeenCalledWith(expect.objectContaining({ capability: NOTE_EDITOR }))
  })

  it('defaults softWrap to true and lineNumbers to false when no settings are stored', () => {
    expect(readMarkdownEditorPreferences()).toStrictEqual({
      softWrap: true,
      lineNumbers: false,
    })
  })

  it('wires plugin settings into open note editors and updates them when settings change', async () => {
    const settings: Record<string, unknown> = {
      softWrap: false,
      lineNumbers: true,
    }
    const listeners = new Map<string, (value: unknown) => void>()
    const context = {
      register: { reader: vi.fn(), tabKind: vi.fn() },
      vault: {},
      settings: {
        get: (key: string) => settings[key],
        onChange: (key: string, handler: (value: unknown) => void) => {
          listeners.set(key, handler)
          return { dispose: () => listeners.delete(key) }
        },
      },
    } as unknown as PluginContext

    expect(readMarkdownEditorPreferences(context.settings.get)).toStrictEqual({
      softWrap: false,
      lineNumbers: true,
    })

    activate(context)

    const wrapper = mount(MarkdownNoteEditor, {
      props: { modelValue: 'hello', filePath: 'test.md' },
      attachTo: document.body,
    })
    await flushPromises()

    const view = EditorView.findFromDOM(wrapper.element.querySelector('.cm-editor') as HTMLElement)
    expect(view).not.toBeNull()
    expect(view!.contentDOM.classList.contains('cm-lineWrapping')).toBe(false)
    expect(view!.dom.querySelector('.cm-lineNumbers')).not.toBeNull()
    expect(view!.state.tabSize).toBe(2)

    settings.softWrap = true
    settings.lineNumbers = false
    listeners.get('softWrap')?.(true)
    listeners.get('lineNumbers')?.(false)
    await flushPromises()

    expect(view!.contentDOM.classList.contains('cm-lineWrapping')).toBe(true)
    expect(view!.dom.querySelector('.cm-lineNumbers')).toBeNull()
    expect(view!.state.tabSize).toBe(2)

    wrapper.unmount()
  })

  it('does not read a tab size from the settings', () => {
    const lookup = (key: string) => (key === 'tab_size' || key === 'tabSize' ? 8 : undefined)
    expect(readMarkdownEditorPreferences(lookup)).toStrictEqual({
      softWrap: true,
      lineNumbers: false,
    })
  })

  it('runs deactivation hook cleanly', () => {
    expect(() => deactivate()).not.toThrow()
  })
})
