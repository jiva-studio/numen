/**
 * Reads one line of a note as the backend does: a list marker, a checkbox, then the text with its
 * todo.txt fields. Offsets are in the line, which is what an editor decorates.
 */

export type FieldKind = 'priority' | 'due' | 'done' | 'recurrence' | 'project' | 'context' | 'tag'

interface TaskField {
  readonly kind: FieldKind
  readonly from: number
  readonly to: number
}

export interface TaskLine {
  readonly markerFrom: number
  readonly textFrom: number
  readonly isDone: boolean
  readonly fields: readonly TaskField[]
}

const MARKER = /^(?:[-*+]|\d{1,9}[.)])[ ]+/
const BOX = /^\[(.)\](?=[ \t]|$)/u
const LEADING_PRIORITY = /^\([A-Z]\)$/
const PREFIXED: readonly (readonly [string, FieldKind])[] = [
  ['pri:', 'priority'],
  ['due:', 'due'],
  ['done:', 'done'],
  ['rec:', 'recurrence'],
]

function skipQuotes(line: string): number {
  let at = 0
  for (;;) {
    const blanks = /^[ \t]*/.exec(line.slice(at))?.[0].length ?? 0
    if (line[at + blanks] !== '>') return at
    at += blanks + 1
  }
}

function maskCode(text: string): boolean[] {
  const masked = new Array<boolean>(text.length + 1).fill(false)
  for (const span of text.matchAll(/(`+)[^`]*?\1/g)) {
    for (let i = span.index; i < span.index + span[0].length; i++) masked[i] = true
  }
  return masked
}

function getKind(word: string, isFirst: boolean): FieldKind | undefined {
  if (isFirst && LEADING_PRIORITY.test(word)) return 'priority'
  const prefixed = PREFIXED.find(
    ([prefix]) => word.startsWith(prefix) && word.length > prefix.length,
  )
  if (prefixed) return prefixed[1]
  if (word.length > 1 && word[0] === '+') return 'project'
  if (word.length > 1 && word[0] === '@') return 'context'
  if (word.length > 1 && word[0] === '#') return 'tag'
  return undefined
}

function readFields(line: string, textFrom: number): TaskField[] {
  const text = line.slice(textFrom)
  const masked = maskCode(text)
  const fields: TaskField[] = []
  let index = 0
  for (const token of text.matchAll(/[^ \t]+/g)) {
    const kind = masked[token.index] ? undefined : getKind(token[0], index === 0)
    if (kind) {
      fields.push({
        kind,
        from: textFrom + token.index,
        to: textFrom + token.index + token[0].length,
      })
    }
    index += 1
  }
  return fields
}

export function readTaskLine(line: string): TaskLine | null {
  const quoted = skipQuotes(line)
  const indent = /^[ \t]*/.exec(line.slice(quoted))?.[0].length ?? 0
  const markerFrom = quoted + indent
  const marker = MARKER.exec(line.slice(markerFrom))
  if (!marker) return null
  const boxFrom = markerFrom + marker[0].length
  const box = BOX.exec(line.slice(boxFrom))
  if (!box || box[1] === ']') return null
  const afterBox = boxFrom + box[0].length
  const gap = /^[ \t]*/.exec(line.slice(afterBox))?.[0].length ?? 0
  const textFrom = afterBox + gap
  if (line.slice(textFrom).trim() === '') return null
  return {
    markerFrom,
    textFrom,
    isDone: box[1] === 'x' || box[1] === 'X',
    fields: readFields(line, textFrom),
  }
}

interface Open {
  readonly char: string
  readonly size: number
}

function readFence(
  line: string,
): { readonly char: string; readonly size: number; readonly rest: string } | undefined {
  const fence = /^ {0,3}(`{3,}|~{3,})(.*)$/.exec(line)
  if (!fence?.[1]) return undefined
  return { char: fence[1][0] ?? '`', size: fence[1].length, rest: fence[2] ?? '' }
}

function isOpening(fence: ReturnType<typeof readFence>): boolean {
  return fence !== undefined && !(fence.char === '`' && fence.rest.includes('`'))
}

function isClosing(fence: ReturnType<typeof readFence>, open: Open): boolean {
  return (
    fence !== undefined &&
    fence.char === open.char &&
    fence.size >= open.size &&
    fence.rest.trim() === ''
  )
}

export function createFenceTracker(): (line: string) => boolean {
  let open: Open | undefined
  return (line) => {
    const fence = readFence(line)
    if (open) {
      if (isClosing(fence, open)) open = undefined
      return true
    }
    if (fence && isOpening(fence)) {
      open = { char: fence.char, size: fence.size }
      return true
    }
    return false
  }
}
