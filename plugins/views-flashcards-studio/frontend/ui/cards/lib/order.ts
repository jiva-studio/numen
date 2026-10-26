/**
 * What every view of a stencil or a deck works in: the order entries stand in,
 * the names they may be given, and the halves a face is written in. No DOM, no
 * measurement, no clock.
 */

/** Where a dragged entry lands: before the entry named, or at the end. */
export type InsertionPoint = string | null

/**
 * The order a dragged entry lands in. The entry is taken out first, so landing
 * before itself, before nothing, or before a name that is not there leaves the
 * order as it was.
 */
export function orderNames(
  names: readonly string[],
  dragEntry: string,
  targetPoint: InsertionPoint,
): readonly string[] {
  if (!names.includes(dragEntry)) return names

  const remaining = names.filter((name) => name !== dragEntry)
  if (targetPoint === null) return [...remaining, dragEntry]

  const before = remaining.indexOf(targetPoint)
  if (before === -1) return names
  return [...remaining.slice(0, before), dragEntry, ...remaining.slice(before)]
}

/**
 * A dragged field may land where it was let go. The first field names every
 * card the stencil cuts, so it stays first: it does not move, and nothing lands
 * above it. A field let go at its current position moves nothing either.
 */
export function canDropField(
  fields: readonly string[],
  dragEntry: string,
  at: InsertionPoint,
): boolean {
  const first = fields[0]
  if (first === undefined) return false
  if (dragEntry === first || at === first) return false
  return dragEntry !== at
}

/** Which way along the order something is dragged by the keyboard. */
export type StepDirection = 'up' | 'down'

/** The direction along the order an arrow drags what is held, and nothing for any other key. */
export const parseStepKey = (key: string): StepDirection | null => {
  if (key === 'ArrowUp') return 'up'
  if (key === 'ArrowDown') return 'down'
  return null
}

/** The keys that drag what is held one place, as a reader is told them. */
export const STEP_KEYS = 'ArrowUp ArrowDown'

/**
 * Where a dragged entry lands one place along the order, and nothing where
 * there is no place that way. Landing before the entry past the next one is
 * what puts it one place further down, the entry being taken out first.
 */
export function getStepInsertionPoint(
  names: readonly string[],
  dragEntry: string,
  direction: StepDirection,
): InsertionPoint | undefined {
  const pos = names.indexOf(dragEntry)
  if (pos === -1) return undefined
  if (direction === 'up') return names[pos - 1]
  if (pos === names.length - 1) return undefined
  return names[pos + 2] ?? null
}

/** The order a dragged field lands in, with the first field remaining where it is. */
export const reorderFields = (
  fields: readonly string[],
  dragEntry: string,
  targetPoint: InsertionPoint,
): readonly string[] =>
  canDropField(fields, dragEntry, targetPoint) ? orderNames(fields, dragEntry, targetPoint) : fields

/** Why a name cannot be used, and nothing where it can. */
export type Objection = 'blank' | 'taken' | 'braced'

export type HeadingObjection = 'blank' | 'taken'

/** What checking a name came back with. */
export interface NameCheckResult<Why extends Objection> {
  /** The name as it would be written, without the space around it. */
  readonly name: string
  /** Why it cannot be used, and nothing where it can. */
  readonly objection: Why | null
}

/**
 * What is wrong with a name. A name is what a slot is written by, so a name
 * carrying a brace cannot be written, and one already taken names two slots.
 */
export function checkFieldName(name: string, names: readonly string[]): NameCheckResult<Objection> {
  const cleanName = name.trim()
  if (cleanName.includes('{') || cleanName.includes('}'))
    return { name: cleanName, objection: 'braced' }
  return checkHeadingName(name, names)
}

/**
 * What is wrong with a name that stands as a heading. It is written nowhere a
 * brace is read, so a brace in it is a character like any other.
 */
export function checkHeadingName(
  name: string,
  names: readonly string[],
): NameCheckResult<HeadingObjection> {
  const cleanName = name.trim()
  if (cleanName === '') return { name: cleanName, objection: 'blank' }
  if (names.some((each) => each.trim() === cleanName))
    return { name: cleanName, objection: 'taken' }
  return { name: cleanName, objection: null }
}

/**
 * The first free name numbered from a stem: `Field 1`, `Field 2`, … The stem
 * alone is not one of them, so every name made this way carries a number.
 */
export function getFreeName(names: readonly string[], stem: string): string {
  const target = new Set(names.map((name) => name.trim()))
  for (let i = 1; ; i += 1) {
    const tried = `${stem} ${i}`
    if (!target.has(tried)) return tried
  }
}

/**
 * A name is compared as written and the first of two stands, so a name a
 * stencil declares twice is one field.
 */
export const getDeclaredFields = (fields: readonly string[]): readonly string[] => [
  ...new Set(fields),
]

/** Which half of a face is drawn. */
export type Half = 'front' | 'back'

/** Both halves, in the order they are drawn. */
export const HALVES: readonly Half[] = ['front', 'back']

/** What is wrong with each of a number of things, under what each is known by. */
export type Problems = ReadonlyMap<string, readonly string[]>

export function createSealedMap<K, V>(): ReadonlyMap<K, V> {
  const empty = new Map<K, V>()
  const refuse = (): never => {
    throw new TypeError('an empty default holds nothing')
  }
  return Object.freeze(Object.assign(empty, { set: refuse, delete: refuse, clear: refuse }))
}
