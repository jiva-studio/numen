import type { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'

/** Whether this editor takes typing at all. */
export const isWritable = (state: EditorState) =>
  !state.readOnly && state.facet(EditorView.editable)
