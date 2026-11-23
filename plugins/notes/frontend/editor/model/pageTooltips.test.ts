import { afterEach, describe, expect, it, vi } from 'vitest'
import { autocompletion, currentCompletions, startCompletion } from '@codemirror/autocomplete'
import { EditorState, StateEffect } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { theme } from '../lib/config/theme'
import { StyleModule } from 'style-mod'
import { createPageTooltips } from './pageTooltips'

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
        theme,
        createPageTooltips(),
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

  it('mount no height for the bare root of the theme, which the container of the tooltips in the page also carries', async () => {
    const view = open(document.body)

    await showList(view)

    const pageStyles = Array.from(document.querySelectorAll('style'))
      .map((one) => one.textContent ?? '')
      .join('')
    expect(pageStyles).toMatch(/\.cm-editor\s*\{[^}]*height:\s*100%/)
    expect(pageStyles).not.toMatch(/\.\u037c\d+\s*\{[^}]*height:/)
  })

  it('mount the styles again only when the set of style modules changes', () => {
    const view = open(document.body)
    const mount = vi.spyOn(StyleModule, 'mount')

    view.dispatch({ changes: { from: 0, insert: 'x' } })
    view.dispatch({ changes: { from: 0, insert: 'y' } })
    expect(mount).not.toHaveBeenCalled()

    const extra = new StyleModule({ '.extra': { color: 'red' } })
    view.dispatch({ effects: StateEffect.appendConfig.of(EditorView.styleModule.of(extra)) })
    const afterChange = mount.mock.calls.length
    expect(afterChange).toBeGreaterThan(0)

    view.dispatch({ changes: { from: 0, insert: 'z' } })
    expect(mount).toHaveBeenCalledTimes(afterChange)
    mount.mockRestore()
  })
})
