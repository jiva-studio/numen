/**
 * What the parser's tags are painted in.
 *
 * Covers the languages a fenced block can be written in. What the markdown
 * itself looks like is decided in `live.ts`, which draws the constructs.
 */
import { HighlightStyle } from '@codemirror/language'
import { tags as t } from '@lezer/highlight'

export const highlighting = HighlightStyle.define([
  {
    tag: [t.keyword, t.modifier, t.controlKeyword, t.operatorKeyword],
    color: 'var(--editor-keyword)',
  },
  {
    tag: [t.name, t.variableName, t.propertyName, t.attributeName],
    color: 'var(--editor-name)',
  },
  { tag: [t.typeName, t.className, t.namespace, t.tagName], color: 'var(--editor-type)' },
  { tag: [t.string, t.special(t.string), t.regexp], color: 'var(--editor-string)' },
  { tag: [t.number, t.bool, t.atom, t.literal], color: 'var(--editor-number)' },
  {
    tag: [t.comment, t.lineComment, t.blockComment, t.docComment],
    color: 'var(--editor-comment)',
    fontStyle: 'italic',
  },
  {
    tag: [t.operator, t.punctuation, t.bracket, t.separator, t.derefOperator],
    color: 'var(--editor-punctuation)',
  },
  { tag: t.meta, color: 'var(--editor-comment)' },
  // A mark shown under the caret is quieter than the words it marks.
  { tag: t.processingInstruction, color: 'var(--editor-mark)' },
  { tag: t.escape, color: 'var(--editor-number)' },
  { tag: t.link, color: 'var(--editor-link)' },
  { tag: t.invalid, color: 'var(--numen-alarm)' },
])
