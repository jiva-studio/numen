import { afterEach, describe, expect, it, vi } from 'vitest'
import { EditorView } from '@codemirror/view'
import { createTranscriptView, type TranscriptViewOptions } from './transcriptView'

const views: { destroy(): void }[] = []

afterEach(() => {
  for (const view of views.splice(0)) view.destroy()
  document.body.innerHTML = ''
})

const open = (options: Partial<TranscriptViewOptions> = {}) => {
  const parent = document.createElement('div')
  document.body.append(parent)
  const onChange = vi.fn()
  const onSave = vi.fn()
  const view = createTranscriptView({
    parent,
    text: 'A bell over the door.\nRain on the awning.',
    isEditable: true,
    label: 'Transcript',
    extensions: [],
    onChange,
    onSave,
    ...options,
  })
  views.push(view)
  const editor = EditorView.findFromDOM(parent.querySelector('.cm-editor') as HTMLElement)
  if (!editor) throw new Error('no editor drawn')
  return { view, editor, parent, onChange, onSave }
}

describe('createTranscriptView', () => {
  it('reports an edit by the person with the whole text', () => {
    const { editor, onChange } = open()

    editor.dispatch({ changes: { from: 0, to: 1, insert: 'One' } })

    expect(onChange).toHaveBeenCalledWith('One bell over the door.\nRain on the awning.')
  })

  it('does not report text set from outside', () => {
    const { view, editor, onChange } = open()

    view.setText('Someone counting change.')

    expect(editor.state.doc.toString()).toBe('Someone counting change.')
    expect(onChange).not.toHaveBeenCalled()
  })

  it('names the text by its label', () => {
    const { parent } = open()

    expect(parent.querySelector('.cm-content')?.getAttribute('aria-label')).toBe('Transcript')
  })

  it('is read only until made editable', () => {
    const { view, editor } = open({ isEditable: false })
    expect(editor.state.readOnly).toBe(true)

    view.setEditable(true)

    expect(editor.state.readOnly).toBe(false)
  })

  it('asks to save on Mod-s', () => {
    const { parent, onSave } = open()
    const content = parent.querySelector('.cm-content') as HTMLElement

    content.dispatchEvent(new KeyboardEvent('keydown', { key: 's', ctrlKey: true, bubbles: true }))

    expect(onSave).toHaveBeenCalledTimes(1)
  })

  it('draws no Markdown syntax', () => {
    const { parent } = open({ text: '# Not a heading' })

    expect(parent.querySelector('.cm-line')?.innerHTML).toBe('# Not a heading')
  })
})
