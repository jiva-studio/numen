import { afterEach, describe, expect, it, vi } from 'vitest'
import { autocompletion, currentCompletions, startCompletion } from '@codemirror/autocomplete'
import { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { pageTooltips } from './pageTooltips'

const views: EditorView[] = []

afterEach(() => {
  views.splice(0).forEach((view) => view.destroy())
})

function open(parent: HTMLElement): EditorView {
  const source = () => ({ from: 0, options: [{ label: 'one' }] })
  const view = new EditorView({
    parent,
    state: EditorState.create({
      doc: 'one',
      extensions: [
        pageTooltips(),
        autocompletion(),
        EditorState.languageData.of(() => [{ autocomplete: source }]),
      ],
    }),
  })
  views.push(view)
  return view
}

async function showList(view: EditorView): Promise<Element | null> {
  startCompletion(view)
  await vi.waitFor(() => expect(currentCompletions(view.state)).toHaveLength(1))
  return document.querySelector('.cm-tooltip-autocomplete')
}

describe('the page tooltips', () => {
  it('draw the completion list in the page, not inside the editor', async () => {
    const view = open(document.body)

    const list = await showList(view)

    expect(list).not.toBeNull()
    expect(view.dom.contains(list)).toBe(false)
  })

  it('mount the styles of an editor that lives in a shadow root in the page', async () => {
    const owner = document.createElement('div')
    document.body.appendChild(owner)
    const inside = owner.attachShadow({ mode: 'open' }).appendChild(document.createElement('div'))
    const view = open(inside)

    await showList(view)

    const pageStyles = Array.from(document.querySelectorAll('style'))
      .map((one) => one.textContent ?? '')
      .join('')
    expect(pageStyles).toContain('cm-tooltip-autocomplete')
  })

  it('keep the list in the one layer of the page that holds the tooltips', async () => {
    const first = await showList(open(document.body))
    const second = await showList(open(document.body))

    const layers = document.querySelectorAll('[data-numen-tooltip-layer]')
    expect(layers).toHaveLength(1)
    expect(layers[0]?.contains(first)).toBe(true)
    expect(layers[0]?.contains(second)).toBe(true)
  })
})
