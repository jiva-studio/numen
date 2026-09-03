/**
 * The marks, drawn as what they mean.
 *
 * A construct is shown as it reads until the person stands in it, and then it
 * is shown as it is written. Standing in it means a selection touches it, so
 * the marks come back under the caret and nowhere else. Nothing here changes
 * the text: the file is what was typed, mark for mark.
 */
import { syntaxTree } from '@codemirror/language'
import { StateField, type EditorState, type Range } from '@codemirror/state'
import {
  Decoration,
  type DecorationSet,
  EditorView,
  ViewPlugin,
  type ViewUpdate,
  type WidgetType,
} from '@codemirror/view'
import type { SyntaxNode, SyntaxNodeRef } from '@lezer/common'
import { wikilinkAt } from '../linking/address'
import { gridOf } from './table'
import { Box, Bullet, Picture, Rule } from './widgets'
import { opening } from './outside'

const HEADING = /^(?:ATX|Setext)Heading([1-6])$/
const BULLET = /^[-+*]$/

const hidden = Decoration.replace({})

const lines = new Map<string, Decoration>()
const named = new Map<string, Decoration>()

const line = (name: string) => {
  const drawn = lines.get(name) ?? Decoration.line({ class: name })
  lines.set(name, drawn)
  return drawn
}

const mark = (name: string) => {
  const drawn = named.get(name) ?? Decoration.mark({ class: name })
  named.set(name, drawn)
  return drawn
}

const childOf = (node: SyntaxNode, name: string) => {
  for (let child = node.firstChild; child; child = child.nextSibling) {
    if (child.name === name) return child
  }
  return null
}

/**
 * What is drawn between `from` and `to`.
 *
 * A widget standing for whole lines has to come from the state; everything
 * else is drawn for the lines on screen. `wants` says which of the two this
 * pass is collecting.
 */
const build = (
  state: EditorState,
  from: number,
  to: number,
  wants: 'blocks' | 'marks',
): DecorationSet => {
  const found: Range<Decoration>[] = []
  const doc = state.doc
  const blocks = wants === 'blocks'

  const standing = (start: number, end: number) =>
    state.selection.ranges.some((range) => range.from <= end && range.to >= start)

  const away = (node: SyntaxNodeRef) => !standing(node.from, node.to)

  const under = (node: SyntaxNodeRef) => {
    const parent = node.node.parent
    return parent ? standing(parent.from, parent.to) : true
  }

  /** A mark and the space it is separated from its content by. */
  const hide = (start: number, end: number) => {
    const after = doc.sliceString(end, end + 1) === ' ' ? end + 1 : end
    if (after > start) found.push(hidden.range(start, after))
  }

  const eachLine = (
    start: number,
    end: number,
    add: (at: number, first: boolean, last: boolean) => void,
  ) => {
    for (let at = start, first = true; ; first = false) {
      const row = doc.lineAt(at)
      const last = row.to >= end
      add(row.from, first, last)
      if (last) break
      at = row.to + 1
    }
  }

  /** A widget standing for whole lines. */
  const block = (node: SyntaxNodeRef, widget: WidgetType) =>
    Decoration.replace({ widget, block: true }).range(
      doc.lineAt(node.from).from,
      doc.lineAt(node.to).to,
    )

  syntaxTree(state).iterate({
    from,
    to,
    enter: (node) => {
      switch (node.name) {
        case 'HorizontalRule':
          if (blocks && away(node)) found.push(block(node, new Rule()))
          return false

        case 'Table':
          if (!away(node)) return !blocks
          if (blocks) found.push(block(node, gridOf(state, node.node)))
          return false
      }

      if (blocks) return true

      const heading = HEADING.exec(node.name)
      if (heading) {
        found.push(line(`cm-heading cm-heading-${heading[1]}`).range(doc.lineAt(node.from).from))
        return true
      }

      switch (node.name) {
        case 'HeaderMark': {
          const parent = node.node.parent
          if (!parent || parent.name.startsWith('Setext')) return false
          if (!under(node)) hide(node.from, node.to)
          return false
        }

        case 'QuoteMark':
          if (!under(node)) hide(node.from, node.to)
          return false

        case 'Blockquote':
          eachLine(node.from, node.to, (at) => found.push(line('cm-quote').range(at)))
          return true

        case 'ListMark': {
          const list = node.node.parent?.parent?.name
          if (list !== 'BulletList') return false
          if (BULLET.test(doc.sliceString(node.from, node.to)))
            found.push(Decoration.replace({ widget: new Bullet() }).range(node.from, node.to))
          return false
        }

        case 'TaskMarker': {
          const done = doc.sliceString(node.from, node.to).toLowerCase() === '[x]'
          const box = new Box(done, !state.readOnly)
          found.push(Decoration.replace({ widget: box }).range(node.from, node.to))
          return false
        }

        case 'FencedCode':
          eachLine(node.from, node.to, (at, first, last) => {
            const edge = `${first ? ' cm-code-first' : ''}${last ? ' cm-code-last' : ''}`
            found.push(line(`cm-code${edge}`).range(at))
          })
          return true

        case 'InlineCode':
          found.push(mark('cm-code-inline').range(node.from, node.to))
          return true

        case 'CodeMark':
          if (node.node.parent?.name === 'InlineCode' && !under(node))
            found.push(hidden.range(node.from, node.to))
          return false

        case 'Emphasis':
          found.push(mark('cm-em').range(node.from, node.to))
          return true

        case 'StrongEmphasis':
          found.push(mark('cm-strong').range(node.from, node.to))
          return true

        case 'Strikethrough':
          found.push(mark('cm-strike').range(node.from, node.to))
          return true

        case 'EmphasisMark':
        case 'StrikethroughMark':
          if (!under(node)) found.push(hidden.range(node.from, node.to))
          return false

        case 'Link':
        case 'Autolink':
          found.push(mark('cm-link').range(node.from, node.to))
          return true

        case 'LinkMark':
        case 'URL':
        case 'LinkTitle':
        case 'LinkLabel':
          if (!under(node)) found.push(hidden.range(node.from, node.to))
          return false

        case 'Image': {
          const address = childOf(node.node, 'URL')
          if (!address || !away(node)) return true
          found.push(
            Decoration.replace({
              widget: new Picture(doc.sliceString(address.from, address.to)),
            }).range(node.from, node.to),
          )
          return false
        }

        default:
          return true
      }
    },
  })

  return Decoration.set(found, true)
}

/** What is drawn inside the lines: the marks, and what stands for them. */
export const marks = (state: EditorState, from: number, to: number) => build(state, from, to, 'marks')

/** What is drawn in place of whole lines: a rule, a table. */
export const blockMarks = (state: EditorState) => build(state, 0, state.doc.length, 'blocks')

/**
 * How far past the lines on screen the marks are drawn, in characters. A pane
 * being resized moves the viewport a line at a time, and marks already drawn
 * answer for the lines it reaches.
 */
const margin = 2000

/** The lines the viewport shows, whole. */
const onScreen = (view: EditorView) => {
  const ranges = view.visibleRanges
  if (!ranges.length) return null
  const first = ranges[0]
  const last = ranges[ranges.length - 1]
  if (!first || !last) return null
  const doc = view.state.doc
  return { from: doc.lineAt(first.from).from, to: doc.lineAt(last.to).to }
}

/** Those lines with a margin of them either side, which is what is drawn for. */
const shown = (view: EditorView) => {
  const seen = onScreen(view)
  if (!seen) return null
  const doc = view.state.doc
  return {
    from: doc.lineAt(Math.max(seen.from - margin, 0)).from,
    to: doc.lineAt(Math.min(seen.to + margin, doc.length)).to,
  }
}

/** A widget standing for whole lines is the state's to hold, not a plugin's. */
export const wholeLines = StateField.define<DecorationSet>({
  create: (state) => blockMarks(state),
  update: (was, transaction) =>
    transaction.docChanged ||
    transaction.selection ||
    syntaxTree(transaction.startState) != syntaxTree(transaction.state)
      ? blockMarks(transaction.state)
      : was,
  provide: (field) => [
    EditorView.decorations.from(field),
    EditorView.atomicRanges.of((view) => view.state.field(field)),
  ],
})

export const live = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet
    /** The span the marks in hand were drawn for. */
    span: { from: number; to: number } | null

    constructor(view: EditorView) {
      this.span = shown(view)
      this.decorations = this.span ? marks(view.state, this.span.from, this.span.to) : Decoration.none
    }

    update(update: ViewUpdate) {
      // A parse finishes in chunks, and the transaction that announces one
      // changes neither the document nor the selection. Past the first chunk
      // the document is drawn as it is written until this is asked.
      const afresh =
        update.docChanged ||
        update.selectionSet ||
        syntaxTree(update.startState) != syntaxTree(update.state)

      // The lines on screen, against the wider span the marks in hand were
      // drawn for. A viewport still inside them is already drawn.
      const seen = onScreen(update.view)
      if (!afresh && this.span && seen && seen.from >= this.span.from && seen.to <= this.span.to) {
        return
      }

      const span = shown(update.view)
      this.span = span
      this.decorations = span ? marks(update.view.state, span.from, span.to) : Decoration.none
    }
  },
  {
    decorations: (plugin) => plugin.decorations,
    // A mark that is not drawn is not a place the caret can be put.
    provide: (plugin) =>
      EditorView.atomicRanges.of((view) => view.plugin(plugin)?.decorations ?? Decoration.none),
  },
)

/** Whether a position stands in code, where a link is an example of one. */
const coded = (node: SyntaxNode): boolean => {
  for (let one: SyntaxNode | null = node; one; one = one.parent) {
    if (one.name === 'FencedCode' || one.name === 'CodeBlock' || one.name === 'InlineCode') {
      return true
    }
  }
  return false
}

/**
 * The address a position in the text stands in, and nothing where it stands in
 * none. A note in brackets is read by the parser every link is read by.
 */
export const addressAt = (state: EditorState, at: number): string | null => {
  const standing = syntaxTree(state).resolveInner(at, 1)
  if (!coded(standing)) {
    const line = state.doc.lineAt(at)
    const wiki = wikilinkAt(line.text, at - line.from)
    if (wiki) return wiki.address
  }

  for (let node: SyntaxNode | null = standing; node; node = node.parent) {
    if (node.name !== 'Link' && node.name !== 'Autolink') continue
    const address = childOf(node, 'URL')
    return address
      ? state.doc.sliceString(address.from, address.to)
      : state.doc.sliceString(node.from + 1, node.to - 1)
  }
  return null
}

/** A drawn link is followed with the platform's modifier held down. */
export const following = EditorView.domEventHandlers({
  mousedown(event, view) {
    if (!event.metaKey && !event.ctrlKey) return false
    const at = view.posAtCoords({ x: event.clientX, y: event.clientY })
    if (at === null) return false

    const address = addressAt(view.state, at)
    if (address === null) return false
    view.state.facet(opening)(address)
    event.preventDefault()
    return true
  },
})
