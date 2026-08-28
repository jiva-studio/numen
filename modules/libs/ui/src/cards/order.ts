/**
 * What every view of a stencil or a deck works in: the order entries stand in,
 * the names they may be given, and the halves a face is written in. No DOM, no
 * measurement, no clock.
 */

/** Where a carried entry lands: before the entry named, or at the end. */
export type Landing = string | null

/**
 * The order a carried entry lands in. The entry is taken out first, so landing
 * before itself, before nothing, or before a name that is not there leaves the
 * order as it was.
 */
export function ordered(
  names: readonly string[],
  carried: string,
  at: Landing,
): readonly string[] {
  if (!names.includes(carried)) return names

  const left = names.filter((name) => name !== carried)
  if (at === null) return [...left, carried]

  const before = left.indexOf(at)
  if (before === -1) return names
  return [...left.slice(0, before), carried, ...left.slice(before)]
}

/**
 * A carried field may land where it was let go. The first field names every
 * card the stencil cuts, so it stays first: it does not move, and nothing lands
 * above it. A field let go where it stands moves nothing either.
 */
export function landing(
  fields: readonly string[],
  carried: string,
  at: Landing,
): boolean {
  const first = fields[0]
  if (first === undefined) return false
  if (carried === first || at === first) return false
  return carried !== at
}

/** Which way along the order something is carried by the keyboard. */
export type Way = 'up' | 'down'

/** The way along the order an arrow carries what is held, and nothing for any other key. */
export const wayOf = (key: string): Way | null =>
  key === 'ArrowUp' ? 'up' : key === 'ArrowDown' ? 'down' : null

/**
 * Where a carried entry lands one place along the order, and nothing where
 * there is no place that way. Landing before the entry past the next one is
 * what puts it one place further down, the entry being taken out first.
 */
export function stepped(
  names: readonly string[],
  carried: string,
  way: Way,
): Landing | undefined {
  const at = names.indexOf(carried)
  if (at === -1) return undefined
  if (way === 'up') return at === 0 ? undefined : (names[at - 1] ?? undefined)
  if (at === names.length - 1) return undefined
  return names[at + 2] ?? null
}

/** The order a carried field lands in, with the first field left where it is. */
export const reordered = (
  fields: readonly string[],
  carried: string,
  at: Landing,
): readonly string[] => (landing(fields, carried, at) ? ordered(fields, carried, at) : fields)

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
 * One line of what was typed. The field a card is named by is written in a
 * heading, so the breaks in it close up.
 */
export const oneLine = (text: string): string => text.replace(/\r\n|[\n\r]/g, ' ')

/**
 * The first free name numbered from a stem: `Field 1`, `Field 2`, … The stem
 * alone is not one of them, so every name made this way carries a number.
 */
export function numbered(taken: readonly string[], stem: string): string {
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
export const declared = (fields: readonly string[]): readonly string[] => [...new Set(fields)]

/** Which half of a face is drawn. */
export type Half = 'front' | 'back'

/** Both halves, in the order they are drawn. */
export const HALVES: readonly Half[] = ['front', 'back']

/** What is wrong with each of a number of things, under what each is known by. */
export type Against = ReadonlyMap<string, readonly string[]>
