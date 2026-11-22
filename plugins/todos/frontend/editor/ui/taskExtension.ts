/**
 * The CodeMirror extension the todos plugin adds to the note editor: a checkbox in place of the
 * list marker and checkbox of a task line, and the fields of the task coloured.
 */
import { EditorView, Decoration, WidgetType, type DecorationSet } from '@codemirror/view'
import { EditorState, RangeSetBuilder, StateField, Prec, type Extension } from '@codemirror/state'
import { editorNotePath } from '@numen/plugin-sdk'
import { findFencedLines, readTaskLine, type FieldKind, type TaskLine } from '../lib/taskLine'

interface TickRequest {
  readonly path: string
  readonly line: number
  readonly raw: string
  readonly isDone: boolean
}

export interface TaskExtensionDeps {
  readonly tick: (request: TickRequest) => Promise<void>
  readonly hideFields: boolean
}

class TaskBox extends WidgetType {
  constructor(
    readonly isDone: boolean,
    readonly deps: TaskExtensionDeps,
  ) {
    super()
  }

  override eq(other: TaskBox) {
    return other.isDone === this.isDone && other.deps === this.deps
  }

  override ignoreEvent() {
    return true
  }

  toDOM(view: EditorView) {
    const box = document.createElement('input')
    box.type = 'checkbox'
    box.className = 'cm-todo-box'
    box.checked = this.isDone
    box.disabled = view.state.readOnly || view.state.facet(editorNotePath) === ''
    box.addEventListener('mousedown', (event) => event.preventDefault())
    box.addEventListener('click', (event) => {
      event.preventDefault()
      const line = view.state.doc.lineAt(view.posAtDOM(box))
      this.deps
        .tick({
          path: view.state.facet(editorNotePath),
          line: line.number,
          raw: line.text,
          isDone: !this.isDone,
        })
        .catch(() => undefined)
    })
    return box
  }
}

const FIELD_CLASSES: Readonly<Record<FieldKind, string>> = {
  priority: 'cm-todo-priority',
  due: 'cm-todo-due',
  done: 'cm-todo-done-on',
  recurrence: 'cm-todo-recurrence',
  project: 'cm-todo-project',
  context: 'cm-todo-context',
  tag: 'cm-todo-tag',
}

const FIELD_MARKS = Object.fromEntries(
  Object.entries(FIELD_CLASSES).map(([kind, className]) => [
    kind,
    Decoration.mark({ class: className }),
  ]),
)
const DONE_LINE = Decoration.line({ class: 'cm-todo-line-done' })
const HIDDEN_FIELD = Decoration.replace({})

function addFields(
  builder: RangeSetBuilder<Decoration>,
  line: { readonly from: number; readonly text: string },
  task: TaskLine,
  isHidden: boolean,
): void {
  let previousEnd = task.textFrom
  for (const field of task.fields) {
    let start = field.from
    while (isHidden && start > previousEnd && /[ \t]/.test(line.text[start - 1] ?? '')) start -= 1
    previousEnd = field.to
    const mark = isHidden ? HIDDEN_FIELD : (FIELD_MARKS[field.kind] as Decoration)
    builder.add(line.from + start, line.from + field.to, mark)
  }
}

function buildTaskDecorations(state: EditorState, deps: TaskExtensionDeps): DecorationSet {
  const selection = state.selection.ranges
  const builder = new RangeSetBuilder<Decoration>()
  const lines = state.doc.toString().split('\n')
  const fenced = findFencedLines(lines)
  let offset = 0
  lines.forEach((text, at) => {
    const from = offset
    const to = from + text.length
    offset += text.length + 1
    if (fenced[at]) return
    const task = readTaskLine(text)
    if (!task) return
    if (task.isDone) builder.add(from, from, DONE_LINE)
    builder.add(
      from + task.markerFrom,
      from + task.textFrom,
      Decoration.replace({ widget: new TaskBox(task.isDone, deps) }),
    )
    const isFocused = selection.some((range) => range.from <= to && range.to >= from)
    addFields(builder, { from, text }, task, deps.hideFields && !isFocused)
  })
  return builder.finish()
}

const theme = EditorView.baseTheme({
  '.cm-todo-box': {
    width: '1em',
    height: '1em',
    verticalAlign: 'middle',
    margin: '0 0.3em 0.15em 0',
    accentColor: 'var(--numen-accent)',
    cursor: 'pointer',
  },
  '.cm-todo-line-done': { opacity: '0.55' },
  '.cm-todo-priority': { color: 'var(--numen-accent)', fontWeight: '600' },
  '.cm-todo-due': { color: 'var(--numen-alarm)' },
  '.cm-todo-done-on': { color: 'var(--numen-hushed)' },
  '.cm-todo-recurrence': { color: 'var(--numen-hushed)' },
  '.cm-todo-project': { color: 'var(--numen-syntax-type)' },
  '.cm-todo-context': { color: 'var(--numen-positive)' },
  '.cm-todo-tag': { color: 'var(--numen-syntax-type)' },
})

export function createTaskExtension(deps: TaskExtensionDeps): Extension {
  const decorations = StateField.define<DecorationSet>({
    create: (state) => buildTaskDecorations(state, deps),
    update: (value, tr) =>
      tr.docChanged || (deps.hideFields && tr.selection)
        ? buildTaskDecorations(tr.state, deps)
        : value,
    provide: (field) => EditorView.decorations.from(field),
  })
  return [Prec.high(decorations), theme]
}
