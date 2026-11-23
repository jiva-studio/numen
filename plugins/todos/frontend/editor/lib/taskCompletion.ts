import type { Text } from '@codemirror/state'
import type { Completion, CompletionContext, CompletionResult } from '@codemirror/autocomplete'
import type { FieldCount, ListFieldsResponse } from '../../types'
import { listDateShortcuts } from './dateShortcuts'
import { createFenceTracker, readTaskLine } from './taskLine'

export interface TaskCompletionDeps {
  readonly getFields: () => ListFieldsResponse
  readonly load?: () => Promise<void>
  readonly getToday: () => string
}

const PRIORITIES = Array.from({ length: 26 }, (_, at) => String.fromCharCode(65 + at))

function isMatch(label: string, typed: string): boolean {
  return label.toLowerCase().startsWith(typed.toLowerCase())
}

function listFieldOptions(
  counts: readonly FieldCount[],
  typed: string,
  type: string,
): Completion[] {
  return counts
    .filter((one) => isMatch(one.name, typed))
    .map((one) => ({ label: one.name, detail: String(one.count), type }))
}

function listDateOptions(today: string, typed: string): Completion[] {
  return listDateShortcuts(today)
    .filter((one) => isMatch(one.label, typed))
    .map((one) => ({ label: one.label, detail: one.date, apply: one.date, type: 'date' }))
}

function listPriorityOptions(typed: string): Completion[] {
  return PRIORITIES.filter((one) => isMatch(one, typed)).map((one) => ({
    label: one,
    type: 'priority',
  }))
}

function listOptions(
  word: string,
  deps: TaskCompletionDeps,
): { skip: number; options: Completion[] } | null {
  const prefixes: readonly (readonly [string, (typed: string) => Completion[]])[] = [
    ['+', (typed) => listFieldOptions(deps.getFields().projects, typed, 'project')],
    ['@', (typed) => listFieldOptions(deps.getFields().contexts, typed, 'context')],
    ['#', (typed) => listFieldOptions(deps.getFields().tags, typed, 'tag')],
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

interface WordAt {
  readonly word: string
  readonly from: number
}

function isFencedAt(doc: Text, lineNumber: number): boolean {
  const isFenced = createFenceTracker()
  let result = false
  for (const text of doc.iterLines(1, lineNumber + 1)) result = isFenced(text)
  return result
}

function findWord(context: CompletionContext): WordAt | null {
  const line = context.state.doc.lineAt(context.pos)
  const task = readTaskLine(line.text)
  if (!task) return null
  const before = line.text.slice(0, context.pos - line.from)
  const wordFrom = before.search(/[^ \t]*$/)
  if (wordFrom < task.textFrom) return null
  if (isFencedAt(context.state.doc, line.number)) return null
  return { word: before.slice(wordFrom), from: line.from + wordFrom }
}

/** Completes the fields of a task line; it answers nowhere else, so `[[` links are left alone. */
export function createTaskCompletion(deps: TaskCompletionDeps) {
  const complete = (context: CompletionContext): CompletionResult | null => {
    const at = findWord(context)
    if (!at) return null
    const found = listOptions(at.word, deps)
    if (!found || found.options.length === 0) return null
    return { from: at.from + found.skip, options: found.options, filter: false }
  }
  const { load } = deps
  if (!load) return complete
  return async (context: CompletionContext): Promise<CompletionResult | null> => {
    const found = complete(context)
    if (found || !/^[+@#]/.test(findWord(context)?.word ?? '')) return found
    await load().catch(() => undefined) // a failed read leaves the options as they are; the next keystroke asks again
    return complete(context)
  }
}
