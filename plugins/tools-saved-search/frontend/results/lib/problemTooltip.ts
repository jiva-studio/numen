/**
 * The problem under the cursor of the query editor, shown in a tooltip below it for as long as the
 * cursor sits in its range, as the hover of the pointer shows it.
 */
import { forEachDiagnostic, type Diagnostic } from '@codemirror/lint'
import { StateField, type EditorState } from '@codemirror/state'
import { showTooltip, type Tooltip } from '@codemirror/view'

const findProblemAtCursor = (state: EditorState): { from: number; message: string } | null => {
  const { head, empty } = state.selection.main
  if (!empty) return null
  let found: { from: number; message: string } | null = null
  forEachDiagnostic(state, (diagnostic: Diagnostic, from: number, to: number) => {
    if (found === null && head >= from && head <= to) found = { from, message: diagnostic.message }
  })
  return found
}

const createTooltip = (state: EditorState): Tooltip | null => {
  const problem = findProblemAtCursor(state)
  if (problem === null) return null
  return {
    pos: problem.from,
    above: false,
    create: () => {
      const dom = document.createElement('div')
      dom.className = 'cm-tooltip-lint cm-problem-at-cursor'
      const line = document.createElement('div')
      line.className = 'cm-diagnostic cm-diagnostic-error'
      line.textContent = problem.message
      dom.append(line)
      return { dom }
    },
  }
}

/** The tooltip of the problem the cursor sits in. */
export const problemAtCursor = StateField.define<Tooltip | null>({
  create: createTooltip,
  update: (tooltip, transaction) =>
    transaction.docChanged || transaction.selection || transaction.effects.length > 0
      ? createTooltip(transaction.state)
      : tooltip,
  provide: (field) => showTooltip.from(field),
})
