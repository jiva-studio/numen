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

  it('defaults softWrap to true and lineNumbers to false when no settings are stored', () => {
    expect(readMarkdownEditorPreferences()).toStrictEqual({
      softWrap: true,
      lineNumbers: false,
      tabSize: 2,
    })
  })

  it('wires plugin settings into open note editors and updates them when settings change', async () => {
    const settings: Record<string, unknown> = {
      soft_wrap: false,
      line_numbers: true,
      tab_size: 4,
    }
    const listeners = new Map<string, (value: unknown) => void>()
    const context = {
      registerReader: vi.fn(),
      registerTabKind: vi.fn(),
      getSetting: (key: string) => settings[key],
      onSettingChange: (key: string, handler: (value: unknown) => void) => {
        listeners.set(key, handler)
        return { dispose: () => listeners.delete(key) }
      },
    } as unknown as PluginContext

    expect(readMarkdownEditorPreferences(context.getSetting)).toStrictEqual({
      softWrap: false,
      lineNumbers: true,
      tabSize: 4,
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
    expect(view!.state.tabSize).toBe(4)

    settings.soft_wrap = true
    settings.line_numbers = false
    settings.tab_size = 2
    listeners.get('soft_wrap')?.(true)
    listeners.get('line_numbers')?.(false)
    listeners.get('tab_size')?.(2)
    await flushPromises()

    expect(view!.contentDOM.classList.contains('cm-lineWrapping')).toBe(true)
    expect(view!.dom.querySelector('.cm-lineNumbers')).toBeNull()
    expect(view!.state.tabSize).toBe(2)

    wrapper.unmount()
  })

  it('runs deactivation hook cleanly', () => {
    expect(() => deactivate()).not.toThrow()
  })
})
