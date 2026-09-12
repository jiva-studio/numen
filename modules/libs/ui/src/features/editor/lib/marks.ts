/**
 * What each mark of the markdown is drawn as.
 *
 * A construct is shown as it reads until the person stands in it, and then it
 * is shown as it is written. Standing in it means a selection touches it, so
 * the marks come back under the caret and nowhere else. Nothing here changes
 * the text: the file is what was typed, mark for mark.
 */
import { syntaxTree } from '@codemirror/language'
import type { EditorState, Range } from '@codemirror/state'
import { Decoration, type DecorationSet, type WidgetType } from '@codemirror/view'
import type { SyntaxNodeRef } from '@lezer/common'
import { childOf } from './address'
import { gridOf } from './grid'
import { Box, Bullet, Picture, Rule } from './widgets'

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

  const isSelected = (start: number, end: number) =>
    state.selection.ranges.some((range) => range.from <= end && range.to >= start)

  const away = (node: SyntaxNodeRef) => !isSelected(node.from, node.to)

  const under = (node: SyntaxNodeRef) => {
    const parent = node.node.parent
    return parent ? isSelected(parent.from, parent.to) : true
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
