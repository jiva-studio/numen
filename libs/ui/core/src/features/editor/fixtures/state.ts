/**
 * A state with the markdown parser in it, parsed to the end.
 *
 * The tests' furniture: they have no view to do the parsing for them. It
 * ships to nobody.
 */
import { markdown } from '@codemirror/lang-markdown'
import { ensureSyntaxTree } from '@codemirror/language'
import { EditorState } from '@codemirror/state'
import { LANGUAGES } from '../config/languages'
import { GFM } from '@lezer/markdown'

export const createState = (doc: string, caret = 0) => {
  const state = EditorState.create({
    doc,
    selection: { anchor: Math.min(caret, doc.length) },
    extensions: [markdown({ extensions: GFM, codeLanguages: LANGUAGES })],
  })
  ensureSyntaxTree(state, doc.length, 5000)
  return state
}
