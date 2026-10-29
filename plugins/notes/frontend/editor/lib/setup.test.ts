import { afterEach, describe, expect, it, vi } from 'vitest'
import { ensureSyntaxTree } from '@codemirror/language'
import { EditorState, type Extension } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import type { Tree } from '@lezer/common'
import { LANGUAGES } from '../config/languages'
import { live } from './live/live'
import { tabKeyBinding, escapeKeyBinding, preview, prose, setup, type Settings } from './setup'

// Nothing here has a size, and the editor measures anyway.
Range.prototype.getClientRects = () =>
  Object.assign([], { item: () => null }) as unknown as DOMRectList
Range.prototype.getBoundingClientRect = () => new DOMRect()

const views: EditorView[] = []

const open = (settings: Settings = {}, doc = 'one') => {
  const view = new EditorView({
    parent: document.body,
    state: EditorState.create({ doc, extensions: [setup(settings)] }),
  })
  views.push(view)
  return view
}

const focus = (view: EditorView) => view.contentDOM.dispatchEvent(new FocusEvent('focus'))

afterEach(() => {
  views.splice(0).forEach((view) => view.destroy())
})

describe('the editor as it is announced', () => {
  it('is named Editor where it is given no name', () => {
    expect(open().contentDOM.getAttribute('aria-label')).toBe('Editor')
  })

  it('is named what it is given', () => {
    expect(open({ name: 'Notes' }).contentDOM.getAttribute('aria-label')).toBe('Notes')
  })

  it('is described by what it is given', () => {
    const view = open({ describedBy: 'hint' })
    expect(view.contentDOM.getAttribute('aria-describedby')).toBe('hint')
  })

  it('is described by nothing where it is given nothing', () => {
    expect(open().contentDOM.hasAttribute('aria-describedby')).toBe(false)
  })

  it('shows the placeholder it is given while it is empty', () => {
    const view = open({ placeholder: 'Write here' }, '')
    expect(view.contentDOM.querySelector('.cm-placeholder')?.textContent).toBe('Write here')
  })
})

describe('live preview', () => {
  it('is on where nothing says otherwise', () => {
    expect(open().plugin(live)).not.toBeNull()
  })

  it('is on where it is requested for', () => {
    expect(open({ isLivePreview: true }).plugin(live)).not.toBeNull()
  })

  it('is off where it is turned off', () => {
    expect(open({ isLivePreview: false }).plugin(live)).toBeNull()
  })

  it('is an extension of its own while on and nothing while off', () => {
    const hasPlugin = (extension: Extension) => {
      const view = new EditorView({
        parent: document.body,
        state: EditorState.create({ doc: 'one', extensions: [extension] }),
      })
      views.push(view)
      return view.plugin(live) !== null
    }
    expect(hasPlugin(preview(true))).toBe(true)
    expect(hasPlugin(preview(false))).toBe(false)
  })
})

describe('the markdown the editor is written in', () => {
  // The tree a state is made with is what its first parse reached in its time,
  // so the whole tree is the one the parse finished.
  const parseProse = (doc: string): Tree => {
    const state = EditorState.create({ doc, extensions: [prose()] })
    const tree = ensureSyntaxTree(state, doc.length, 5000)
    if (!tree) throw new Error('the parse did not finish in time')
    return tree
  }

  const getNames = (tree: Tree) => {
    const found = new Set<string>()
    tree.iterate({ enter: (node) => void found.add(node.name) })
    return found
  }

  it('reads a table', () => {
    expect(getNames(parseProse('| a | b |\n|---|---|\n| c | d |\n'))).toContain('Table')
  })

  it('reads a fenced block in the language named after the fence', async () => {
    const javascript = LANGUAGES.find((language) => language.name === 'javascript')
    await javascript?.load()
    const doc = '```js\nlet total = 1\n```\n'
    expect(parseProse(doc).resolveInner(doc.indexOf('total'), 1).name).not.toBe('CodeText')
  })
})

describe('Tab', () => {
  const tab = (view: EditorView) => tabKeyBinding.run?.(view) ?? false

  it('indents while the editor holds it', () => {
    const view = open({}, 'one')
    expect(tab(view)).toBe(true)
    expect(view.state.doc.toString()).not.toBe('one')
  })

  it('is still retained after the text changed', () => {
    const view = open({}, 'one')
    view.dispatch({ changes: { from: 3, insert: ' two' } })
    expect(tab(view)).toBe(true)
  })

  it('walks on after Escape handed it to the page', () => {
    const view = open({}, 'one')
    expect(escapeKeyBinding.run?.(view)).toBe(false)
    expect(tab(view)).toBe(false)
    expect(view.state.doc.toString()).toBe('one')
  })

  it('stays with the page through changes to the text', () => {
    const view = open({}, 'one')
    escapeKeyBinding.run?.(view)
    view.dispatch({ changes: { from: 3, insert: ' two' } })
    expect(tab(view)).toBe(false)
  })

  it('goes back to the editor when the keyboard arrives', () => {
    const view = open({}, 'one')
    escapeKeyBinding.run?.(view)
    focus(view)
    expect(tab(view)).toBe(true)
  })

  it('leaves the answer to the focus to whatever else answers it', () => {
    const observed = vi.fn<(event: Event) => void>()
    const view = open({ extensions: EditorView.domEventHandlers({ focus: observed }) })
    focus(view)
    expect(observed).toHaveBeenCalledTimes(1)
  })

  it('makes no change of its own while the editor already holds it', () => {
    const changes: number[] = []
    const view = open({
      extensions: EditorView.updateListener.of((update) => {
        changes.push(update.transactions.filter((one) => one.effects.length > 0).length)
      }),
    })
    focus(view)
    expect(changes.reduce((sum, count) => sum + count, 0)).toBe(0)
  })
})

describe('Shift-Tab', () => {
  const dedent = (view: EditorView) => tabKeyBinding.shift?.(view) ?? false

  it('takes an indent back while the editor holds Tab', () => {
    const view = open({}, '  one')
    expect(dedent(view)).toBe(true)
    expect(view.state.doc.toString()).toBe('one')
  })

  it('walks on after Escape handed Tab to the page', () => {
    const view = open({}, '  one')
    escapeKeyBinding.run?.(view)
    expect(dedent(view)).toBe(false)
    expect(view.state.doc.toString()).toBe('  one')
  })
})

describe('Escape', () => {
  it('passes Tab over once however often it is pressed', () => {
    const handed: number[] = []
    const view = open({
      extensions: EditorView.updateListener.of((update) => {
        handed.push(update.transactions.filter((one) => one.effects.length > 0).length)
      }),
    })
    escapeKeyBinding.run?.(view)
    escapeKeyBinding.run?.(view)
    expect(handed.reduce((sum, count) => sum + count, 0)).toBe(1)
  })
})

describe('an editor with no placeholder', () => {
  it('shows nothing in its place', () => {
    const view = open({}, '')
    expect(view.contentDOM.querySelector('.cm-placeholder')).toBeNull()
  })
})

describe('editor preferences in setup', () => {
  it('wraps lines by default and disables wrapping when softWrap is false', () => {
    expect(open({}).contentDOM.classList.contains('cm-lineWrapping')).toBe(true)
    expect(open({ softWrap: false }).contentDOM.classList.contains('cm-lineWrapping')).toBe(false)
  })

  it('shows line numbers only when lineNumbers is true', () => {
    expect(open({}).dom.querySelector('.cm-lineNumbers')).toBeNull()
    expect(open({ lineNumbers: true }).dom.querySelector('.cm-lineNumbers')).not.toBeNull()
  })

  it('numbers horizontal rule block widgets continuously with text lines and themes the gutter', () => {
    const view = open({ lineNumbers: true }, '---\ntype: stencil\n---')
    const numbers = Array.from(
      view.dom.querySelectorAll<HTMLElement>('.cm-lineNumbers .cm-gutterElement'),
    )
      .filter((el) => el.style.visibility !== 'hidden')
      .map((el) => el.textContent?.trim())
      .filter(Boolean)
    expect(numbers).toEqual(['1', '2', '3'])

    const gutters = view.dom.querySelector<HTMLElement>('.cm-gutters')!
    expect(getComputedStyle(gutters).backgroundColor).toBe('rgba(0, 0, 0, 0)')
  })

  it('attaches heading and code-block alignment classes to line number gutter elements', () => {
    const view = open({ lineNumbers: true }, '# First\n\n## Second\n\n```ts\nconst x = 1\n```')
    const elements = Array.from(
      view.dom.querySelectorAll<HTMLElement>('.cm-lineNumbers .cm-gutterElement'),
    ).filter((el) => el.style.visibility !== 'hidden')

    expect(elements[0]?.classList.contains('cm-gutter-heading-1')).toBe(true)
    expect(elements[0]?.classList.contains('cm-gutter-gap')).toBe(false)
    expect(elements[2]?.classList.contains('cm-gutter-heading-2')).toBe(true)
    expect(elements[2]?.classList.contains('cm-gutter-gap')).toBe(true)
    expect(elements[4]?.classList.contains('cm-gutter-code-first')).toBe(true)
  })

  it('indents with the configured tabSize', () => {
    const view = open({ tabSize: 4 }, 'one')
    expect(view.state.tabSize).toBe(4)
    expect(tabKeyBinding.run?.(view)).toBe(true)
    expect(view.state.doc.toString()).toBe('    one')
  })
})
