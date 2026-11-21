import { afterEach, describe, expect, it } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { EditorView } from '@codemirror/view'
import QueryEditor from '../ui/QueryEditor.vue'

let wrapper: VueWrapper | null = null

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
})

const open = (query: string) => {
  wrapper = mount(QueryEditor, {
    attachTo: document.body,
    props: { id: 'query', query, problems: [], functions: [] },
  })
  const dom = wrapper.element.querySelector('.cm-editor')
  const view = dom ? EditorView.findFromDOM(dom as HTMLElement) : null
  if (!view) throw new Error('the editor drew no view')
  return view
}

describe('the query of the editor', () => {
  it('follows a query changed outside', async () => {
    const view = open('words("gale")')

    await wrapper?.setProps({ query: 'words("calm")' })

    expect(view.state.doc.toString()).toBe('words("calm")')
  })

  it('keeps the text being typed', async () => {
    const view = open('words("gale")')
    view.dispatch({ changes: { from: 0, insert: ' ' }, userEvent: 'input.type' })

    await wrapper?.setProps({ query: 'words("calm")' })

    expect(view.state.doc.toString()).toBe(' words("gale")')
  })
})

describe('the caret of the query editor', () => {
  it('is drawn by the editor, in step with the text', () => {
    const view = open('words("gale")')

    expect(view.dom.querySelector('.cm-cursorLayer')).not.toBeNull()
  })

  it('stays at the insertion point as the characters before it are deleted', () => {
    const view = open('words("gale")')
    view.dispatch({ selection: { anchor: 11 } })

    for (let step = 0; step < 3; step += 1) {
      const head = view.state.selection.main.head
      view.dispatch({ changes: { from: head - 1, to: head }, userEvent: 'delete.backward' })
    }

    expect(view.state.doc.toString()).toBe('words("g")')
    expect(view.state.selection.main.head).toBe(8)
  })
})
