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
  dragged: string,
  at: InsertionPoint,
): readonly string[] {
  if (!names.includes(dragged)) return names

  const left = names.filter((name) => name !== dragged)
  if (at === null) return [...left, dragged]

  const before = left.indexOf(at)
  if (before === -1) return names
  return [...left.slice(0, before), dragged, ...left.slice(before)]
}

/**
 * A dragged field may land where it was let go. The first field names every
 * card the stencil cuts, so it stays first: it does not move, and nothing lands
 * above it. A field let go where it stands moves nothing either.
 */
export function landing(
  fields: readonly string[],
  dragged: string,
  at: InsertionPoint,
): boolean {
  const first = fields[0]
  if (first === undefined) return false
  if (dragged === first || at === first) return false
  return dragged !== at
}

/** Which way along the order something is dragged by the keyboard. */
export type StepDirection = 'up' | 'down'

/** The direction along the order an arrow drags what is held, and nothing for any other key. */
export const directionOf = (key: string): StepDirection | null =>
  key === 'ArrowUp' ? 'up' : key === 'ArrowDown' ? 'down' : null

/** The keys that drag what is held one place, as a reader is told them. */
export const STEP_KEYS = 'ArrowUp ArrowDown'

/**
 * Where a dragged entry lands one place along the order, and nothing where
 * there is no place that way. Landing before the entry past the next one is
 * what puts it one place further down, the entry being taken out first.
 */
export function getStepLanding(
  names: readonly string[],
  dragged: string,
  direction: StepDirection,
): InsertionPoint | undefined {
  const at = names.indexOf(dragged)
  if (at === -1) return undefined
  if (direction === 'up') return at === 0 ? undefined : (names[at - 1] ?? undefined)
  if (at === names.length - 1) return undefined
  return names[at + 2] ?? null
}

/** The order a dragged field lands in, with the first field left where it is. */
export const reorderFields = (
  fields: readonly string[],
  dragged: string,
  at: InsertionPoint,
): readonly string[] => (landing(fields, dragged, at) ? orderNames(fields, dragged, at) : fields)

/** Why a name cannot be used, and nothing where it can. */
export type Objection = 'blank' | 'taken' | 'braced'

/**
 * What is wrong with a name. A name is what a slot is written by, so a name
 * carrying a brace cannot be written, and one already taken names two slots.
 */
export function objection(name: string, taken: readonly string[]): Objection | null {
  const said = name.trim()
  if (said.includes('{') || said.includes('}')) return 'braced'
  return heading(name, taken)
}

/** Why a name written as a heading and in no slot is refused. */
export type Refusal = 'blank' | 'taken'

/**
 * What is wrong with a name that stands as a heading. It is written nowhere a
 * brace is read, so a brace in it is a character like any other.
 */
export function heading(name: string, taken: readonly string[]): Refusal | null {
  const said = name.trim()
  if (said === '') return 'blank'
  if (taken.some((each) => each.trim() === said)) return 'taken'
  return null
}

/**
 * The first free name numbered from a stem: `Field 1`, `Field 2`, … The stem
 * alone is not one of them, so every name made this way carries a number.
 */
export function getFreeName(taken: readonly string[], stem: string): string {
  const held = new Set(taken.map((name) => name.trim()))
  for (let at = 1; ; at += 1) {
    const tried = `${stem} ${at}`
    if (!held.has(tried)) return tried
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

/**
 * A map of nothing that stays a map of nothing. An empty default stands for
 * every caller at once, so putting anything into it is refused.
 */
export function createSealedMap<K, V>(): ReadonlyMap<K, V> {
  const empty = new Map<K, V>()
  const refuse = (): never => {
    throw new TypeError('an empty default holds nothing')
  }
  return Object.freeze(Object.assign(empty, { set: refuse, delete: refuse, clear: refuse }))
}
