/**
 * Notes follows the editor settings of the application: it draws no editor until the window has
 * loaded them, then wraps lines and numbers them as they say, and follows a change in an editor
 * that is open. The context is the one a window gives; the editors are the real ones.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { Component } from 'vue'
import { EditorView } from '@codemirror/view'
import { NOTE_EDITOR, type HostServices, type ViewRegistration } from '@numen/plugin-sdk'
import { activate, MarkdownNoteEditor } from './index'
import { createEditorSettingsContext } from './testing/editorSettings'

let wrapper: VueWrapper | null = null

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
})

const createNotes = () => ({
  read: vi.fn(() => Promise.resolve({ body: 'A bell over the door.\n' })),
  write: vi.fn(() => Promise.resolve()),
})

const findEditor = (): EditorView | null => {
  const element = wrapper?.element.querySelector('.cm-editor')
  return element ? EditorView.findFromDOM(element as HTMLElement) : null
}

const getEditor = (): EditorView => {
  const view = findEditor()
  if (!view) throw new Error('no editor drawn')
  return view
}

const isWrapped = (view: EditorView) => view.contentDOM.classList.contains('cm-lineWrapping')
const hasNumbers = (view: EditorView) => view.dom.querySelector('.cm-lineNumbers') !== null

const openTab = async () => {
  wrapper = mount(MarkdownNoteEditor, {
    props: { modelValue: 'hello', filePath: 'test.md' },
    attachTo: document.body,
  })
  await flushPromises()
}

describe('the editor of a note tab', () => {
  it('is not created while the editor settings are undefined', async () => {
    const { context } = createEditorSettingsContext(undefined)
    activate(context, { vault: {} } as unknown as HostServices)

    await openTab()

    expect(findEditor()).toBeNull()
  })

  it('is created once the editor settings arrive', async () => {
    const { context, set } = createEditorSettingsContext(undefined)
    activate(context, { vault: {} } as unknown as HostServices)
    await openTab()

    set({ lineNumbers: false, softWrap: true })
    await flushPromises()

    expect(findEditor()).not.toBeNull()
    expect(getEditor().state.doc.toString()).toBe('hello')
  })

  it('is created at once where the window already holds the settings', async () => {
    const { context } = createEditorSettingsContext({ lineNumbers: false, softWrap: true })
    activate(context, { vault: {} } as unknown as HostServices)

    await openTab()

    expect(findEditor()).not.toBeNull()
  })

  it('wraps lines and shows no numbers when the settings say wrap and no numbers', async () => {
    const { context } = createEditorSettingsContext({ lineNumbers: false, softWrap: true })
    activate(context, { vault: {} } as unknown as HostServices)

    await openTab()

    expect(isWrapped(getEditor())).toBe(true)
    expect(hasNumbers(getEditor())).toBe(false)
  })

  it('leaves lines long and numbers them when the settings say no wrap and numbers', async () => {
    const { context } = createEditorSettingsContext({ lineNumbers: true, softWrap: false })
    activate(context, { vault: {} } as unknown as HostServices)

    await openTab()

    expect(isWrapped(getEditor())).toBe(false)
    expect(hasNumbers(getEditor())).toBe(true)
  })

  it('switches the wrap and the numbers of the open editor when the settings change, and keeps its text', async () => {
    const { context, set } = createEditorSettingsContext({ lineNumbers: true, softWrap: false })
    activate(context, { vault: {} } as unknown as HostServices)
    await openTab()
    const view = getEditor()
    expect([isWrapped(view), hasNumbers(view)]).toStrictEqual([false, true])

    set({ lineNumbers: false, softWrap: true })
    await flushPromises()

    expect(findEditor()).toBe(view)
    expect([isWrapped(view), hasNumbers(view)]).toStrictEqual([true, false])
    expect(view.state.doc.toString()).toBe('hello')
  })

  it('switches the wrap alone and leaves the numbers as they were', async () => {
    const { context, set } = createEditorSettingsContext({ lineNumbers: true, softWrap: true })
    activate(context, { vault: {} } as unknown as HostServices)
    await openTab()

    set({ lineNumbers: true, softWrap: false })
    await flushPromises()

    expect([isWrapped(getEditor()), hasNumbers(getEditor())]).toStrictEqual([false, true])
  })

  it('takes the settings of the activation, not those of an earlier one', async () => {
    const first = createEditorSettingsContext({ lineNumbers: true, softWrap: true })
    activate(first.context, { vault: {} } as unknown as HostServices)
    const second = createEditorSettingsContext(undefined)
    activate(second.context, { vault: {} } as unknown as HostServices)

    await openTab()

    expect(findEditor()).toBeNull()
  })

  it('reads nothing from the settings of the plugin: a value stored under the old keys changes nothing', async () => {
    const { context } = createEditorSettingsContext({ lineNumbers: false, softWrap: true })
    const asked: string[] = []
    const followed: string[] = []
    Object.assign(context.settings, {
      get: (key: string) => {
        asked.push(key)
        return { softWrap: false, lineNumbers: true, tabSize: 8, tab_size: 8 }[key]
      },
      onChange: (key: string) => {
        followed.push(key)
        return { dispose: () => undefined }
      },
    })
    activate(context, { vault: {} } as unknown as HostServices)

    await openTab()

    const view = getEditor()
    expect([isWrapped(view), hasNumbers(view)]).toStrictEqual([true, false])
    expect(view.state.tabSize).toBe(2)
    expect(asked).toStrictEqual([])
    expect(followed).toStrictEqual([])
  })
})

describe('the editor of the note-editor view', () => {
  const register = (notes = createNotes()) => {
    const settings = createEditorSettingsContext(undefined)
    activate(settings.context, { vault: { notes } } as unknown as HostServices)
    const view = settings.views.find((one) => one.capability === NOTE_EDITOR)
    if (!view) throw new Error('no note-editor view registered')
    return { ...settings, view, notes }
  }

  const open = async (view: ViewRegistration) => {
    wrapper = mount(view.component as Component, {
      props: { context: { path: 'Shed.md' } },
      attachTo: document.body,
    })
    await flushPromises()
  }

  it('is not created while the editor settings are undefined', async () => {
    const { view } = register()

    await open(view)

    expect(findEditor()).toBeNull()
  })

  it('is created, with the note in it, once the settings arrive', async () => {
    const { view, set } = register()
    await open(view)

    set({ lineNumbers: true, softWrap: false })
    await flushPromises()

    expect(getEditor().state.doc.toString()).toBe('A bell over the door.\n')
    expect([isWrapped(getEditor()), hasNumbers(getEditor())]).toStrictEqual([false, true])
  })

  it('follows a change of the settings without being opened again', async () => {
    const { view, set } = register()
    await open(view)
    set({ lineNumbers: true, softWrap: false })
    await flushPromises()
    const editor = getEditor()

    set({ lineNumbers: false, softWrap: true })
    await flushPromises()

    expect(findEditor()).toBe(editor)
    expect([isWrapped(editor), hasNumbers(editor)]).toStrictEqual([true, false])
  })
})
