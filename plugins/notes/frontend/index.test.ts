import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { EditorView } from '@codemirror/view'
import { NOTE_EDITOR, type PluginContext } from '@numen/plugin-sdk'
import { tabKeyBinding } from './editor/model/setup'
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
    expect(readMarkdownEditorPreferences()).toMatchObject({
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

    expect(readMarkdownEditorPreferences(context.settings.get)).toMatchObject({
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

  it('runs deactivation hook cleanly', () => {
    expect(() => deactivate()).not.toThrow()
  })
})

describe('the editor settings in an open note editor', () => {
  const open = async (stored: Record<string, unknown> = {}, text = 'one\ntwo\nthree') => {
    const settings: Record<string, unknown> = { ...stored }
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
    activate(context)

    const wrapper = mount(MarkdownNoteEditor, {
      props: { modelValue: text, filePath: 'test.md' },
      attachTo: document.body,
    })
    await flushPromises()
    const view = EditorView.findFromDOM(
      wrapper.element.querySelector('.cm-editor') as HTMLElement,
    ) as EditorView
    const change = async (key: string, value: unknown) => {
      settings[key] = value
      listeners.get(key)?.(value)
      await flushPromises()
    }
    return { view, wrapper, change }
  }
  const tab = (view: EditorView) => tabKeyBinding.run?.(view) ?? false
  // The attribute selector asks the element whether its style has the text, without reading the style.
  const hasStyle = (view: EditorView, text: string) => view.dom.matches(`[style*='${text}']`)

  it('opens with Tab of 2 spaces, no active line, no font variable', async () => {
    const { view, wrapper } = await open()

    tab(view)
    expect(view.state.doc.toString().startsWith('  one')).toBe(true)
    expect(view.state.doc.toString().startsWith('   one')).toBe(false)
    expect(view.dom.querySelector('.cm-activeLine')).toBeNull()
    expect(hasStyle(view, '--editor-font')).toBe(false)
    wrapper.unmount()
  })

  it('follows a change of the tabSize setting in an open editor', async () => {
    const { view, wrapper, change } = await open({ tabSize: 4 })
    expect(view.state.tabSize).toBe(4)

    await change('tabSize', 3)
    tab(view)

    expect(view.state.tabSize).toBe(3)
    expect(view.state.doc.toString().startsWith('   one')).toBe(true)
    expect(view.state.doc.toString().startsWith('    one')).toBe(false)
    wrapper.unmount()
  })

  it('starts and stops marking the active line when the setting changes', async () => {
    const { view, wrapper, change } = await open()
    expect(view.dom.querySelector('.cm-activeLine')).toBeNull()

    await change('highlightActiveLine', true)
    expect(view.dom.querySelector('.cm-activeLine')).not.toBeNull()

    await change('highlightActiveLine', false)
    expect(view.dom.querySelector('.cm-activeLine')).toBeNull()
    wrapper.unmount()
  })

  it('sets the font of an open editor when the setting changes and clears it when the setting is empty', async () => {
    const { view, wrapper, change } = await open()

    await change('fontFamily', 'Fira Code')
    expect(hasStyle(view, '--editor-font')).toBe(true)
    expect(hasStyle(view, '"Fira Code"')).toBe(true)

    await change('fontFamily', '')
    expect(hasStyle(view, '--editor-font')).toBe(false)
    wrapper.unmount()
  })

  it('never puts a font value that breaks out of the style into the editor', async () => {
    const { view, wrapper, change } = await open()

    await change('fontFamily', 'serif; position: fixed')

    expect(hasStyle(view, 'position')).toBe(false)
    expect(hasStyle(view, '--editor-font')).toBe(false)
    wrapper.unmount()
  })

  it('sets the line height of an open editor when the setting changes', async () => {
    const { view, wrapper, change } = await open()

    await change('lineHeight', 1.8)

    expect(hasStyle(view, '--editor-line-height: 1.8')).toBe(true)
    wrapper.unmount()
  })

  it('keeps the font variable when the line height changes', async () => {
    const { view, wrapper, change } = await open({ fontFamily: 'serif' })

    await change('lineHeight', 2)

    expect(hasStyle(view, '--editor-font: serif')).toBe(true)
    expect(hasStyle(view, '--editor-line-height: 2')).toBe(true)
    wrapper.unmount()
  })

  it('keeps the same editor view and text when settings change', async () => {
    const { view, wrapper, change } = await open({}, 'one\ntwo')
    view.dispatch({ selection: { anchor: 5 } })

    await change('tabSize', 4)
    await change('highlightActiveLine', true)
    await change('fontFamily', 'serif')
    await change('lineHeight', 2)

    const again = EditorView.findFromDOM(wrapper.element.querySelector('.cm-editor') as HTMLElement)
    expect(again).toBe(view)
    expect(view.state.doc.toString()).toBe('one\ntwo')
    expect(view.state.selection.main.head).toBe(5)
    wrapper.unmount()
  })
})
