import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { Component } from 'vue'
import { EditorView } from '@codemirror/view'
import { NOTE_EDITOR, type HostServices } from '@numen/plugin-sdk'
import { createNoteEditorView } from './noteEditorView'
import { activate } from '../index'
import { createEditorSettingsContext } from '../testing/editorSettings'

const reported = (text: string): Error => Object.assign(new Error(text), { name: 'ReportedError' })

const loadEditorSettings = () =>
  activate(createEditorSettingsContext({ lineNumbers: false, softWrap: true }).context, {
    vault: {},
    ui: {},
  } as unknown as HostServices)

beforeEach(loadEditorSettings)

let wrapper: VueWrapper | null = null

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
})

const createNotes = (body = 'A bell over the door.\n') => ({
  read: vi.fn(() => Promise.resolve({ body, at: 'first' })),
  write: vi.fn((_path: string, _text: string, _observed: { body: string; at: string }) =>
    Promise.resolve('next'),
  ),
})

const open = async (notes: ReturnType<typeof createNotes>, path = 'Shed.md') => {
  const view = createNoteEditorView(notes)
  wrapper = mount(view.component as Component, {
    props: { context: { path } },
    attachTo: document.body,
  })
  await flushPromises()
  const editor = EditorView.findFromDOM(wrapper.element.querySelector('.cm-editor') as HTMLElement)
  if (!editor) throw new Error('no editor drawn')
  return editor
}

describe('createNoteEditorView', () => {
  it('declares the note-editor capability', () => {
    expect(createNoteEditorView(createNotes()).capability).toBe(NOTE_EDITOR)
  })

  it('shows the note at the path of its context', async () => {
    const notes = createNotes()

    const editor = await open(notes)

    expect(notes.read).toHaveBeenCalledWith('Shed.md')
    expect(editor.state.doc.toString()).toBe('A bell over the door.\n')
  })

  it('writes an edit back to the note', async () => {
    const notes = createNotes()
    const editor = await open(notes)

    editor.dispatch({ changes: { from: 0, insert: 'Rain. ' } })
    await flushPromises()

    expect(notes.write).toHaveBeenLastCalledWith('Shed.md', 'Rain. A bell over the door.\n', {
      body: 'A bell over the door.\n',
      at: 'first',
    })
  })

  it('keeps an external edit on disk and shows the conflict alongside the local draft', async () => {
    let disk = 'original'
    const notes = createNotes(disk)
    notes.write.mockImplementation(async (_path, text, observed) => {
      if (observed.body !== disk) throw reported('The note changed on disk.')
      disk = text
      return 'next'
    })
    const editor = await open(notes)
    disk = 'external edit'
    editor.dispatch({ changes: { from: 0, to: editor.state.doc.length, insert: 'local draft' } })
    await flushPromises()
    expect(disk).toBe('external edit')
    expect(editor.state.doc.toString()).toBe('local draft')
    expect(wrapper!.get('[role="alert"]').text()).toContain('changed on disk')
  })

  it('uses the fingerprint returned by the preceding save for an edit typed in flight', async () => {
    const notes = createNotes('a')
    let finish!: (at: string) => void
    notes.write.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve
        }),
    )
    const editor = await open(notes)
    editor.dispatch({ changes: { from: 1, insert: 'b' } })
    editor.dispatch({ changes: { from: 2, insert: 'c' } })
    expect(notes.write).toHaveBeenCalledTimes(1)
    finish('second')
    await flushPromises()
    expect(notes.write).toHaveBeenLastCalledWith('Shed.md', 'abc', {
      body: 'ab',
      at: 'second',
    })
  })

  it('shows why the note could not be read', async () => {
    const notes = createNotes()
    notes.read.mockResolvedValueOnce({ body: '', error: 'the note is gone' } as never)

    await open(notes)

    expect(wrapper!.get('[role="alert"]').text()).toContain('the note is gone')
  })
})
