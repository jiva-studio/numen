import type { Completion, CompletionContext, CompletionResult } from '@codemirror/autocomplete'
import type { FieldCount, ListFieldsResponse } from '../../types'
import { listDateShortcuts } from './dateShortcuts'
import { findFencedLines, readTaskLine } from './taskLine'

export interface TaskCompletionDeps {
  readonly getFields: () => ListFieldsResponse
  readonly getToday: () => string
}

const PRIORITIES = Array.from({ length: 26 }, (_, at) => String.fromCharCode(65 + at))

function matches(label: string, typed: string): boolean {
  return label.toLowerCase().startsWith(typed.toLowerCase())
}

function listFieldOptions(counts: readonly FieldCount[], typed: string): Completion[] {
  return counts
    .filter((one) => matches(one.name, typed))
    .map((one) => ({ label: one.name, detail: String(one.count), type: 'keyword' }))
}

function listDateOptions(today: string, typed: string): Completion[] {
  return listDateShortcuts(today)
    .filter((one) => matches(one.label, typed))
    .map((one) => ({ label: one.label, detail: one.date, apply: one.date, type: 'constant' }))
}

function listPriorityOptions(typed: string): Completion[] {
  return PRIORITIES.filter((one) => matches(one, typed)).map((one) => ({
    label: one,
    type: 'enum',
  }))
}

function listOptions(
  word: string,
  deps: TaskCompletionDeps,
): { skip: number; options: Completion[] } | null {
  const prefixes: readonly (readonly [string, (typed: string) => Completion[]])[] = [
    ['+', (typed) => listFieldOptions(deps.getFields().projects, typed)],
    ['@', (typed) => listFieldOptions(deps.getFields().contexts, typed)],
    ['due:', (typed) => listDateOptions(deps.getToday(), typed)],
    ['done:', (typed) => listDateOptions(deps.getToday(), typed)],
    ['pri:', listPriorityOptions],
  ]
  for (const [prefix, list] of prefixes) {
    if (word.startsWith(prefix))
      return { skip: prefix.length, options: list(word.slice(prefix.length)) }
  }
  return null
}

/** Completes the fields of a task line; it answers nowhere else, so `[[` links are left alone. */
export function createTaskCompletion(deps: TaskCompletionDeps) {
  return (context: CompletionContext): CompletionResult | null => {
    const line = context.state.doc.lineAt(context.pos)
    const task = readTaskLine(line.text)
    if (!task) return null
    const before = line.text.slice(0, context.pos - line.from)
    const wordFrom = before.search(/[^ \t]*$/)
    if (wordFrom < task.textFrom) return null
    const lines = context.state.doc.sliceString(0, line.to).split('\n')
    if (findFencedLines(lines)[lines.length - 1]) return null
    const found = listOptions(before.slice(wordFrom), deps)
    if (!found || found.options.length === 0) return null
    return {
      from: line.from + wordFrom + found.skip,
      options: found.options,
      filter: false,
    }
  }
}
