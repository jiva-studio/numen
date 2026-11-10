import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { Component } from 'vue'
import { EditorView } from '@codemirror/view'
import { NOTE_EDITOR } from '@numen/plugin-sdk'
import { createNoteEditorView } from './noteEditorView'

let wrapper: VueWrapper | null = null

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
})

const createNotes = (body = 'A bell over the door.\n') => ({
  read: vi.fn(() => Promise.resolve({ body })),
  write: vi.fn(() => Promise.resolve()),
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

    expect(notes.write).toHaveBeenLastCalledWith('Shed.md', 'Rain. A bell over the door.\n')
  })

  it('shows why the note could not be read', async () => {
    const notes = createNotes()
    notes.read.mockResolvedValueOnce({ body: '', error: 'the note is gone' } as never)

    await open(notes)

    expect(wrapper!.get('[role="alert"]').text()).toContain('the note is gone')
  })
})
