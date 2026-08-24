/**
 * What a palette is, as plain values. No DOM, no measurement, no clock.
 */

/** A run of a line, by where it begins and where it ends. */
export interface PaletteSpan {
  readonly from: number
  readonly to: number
}

/** One run of a line, and whether it is why the item is here. */
export interface PalettePart {
  readonly text: string
  readonly hit: boolean
}

/**
 * One thing that can be done to an item, named by whoever offers it. The first
 * is what Enter reaches and the second what Shift and Enter reach; every one of
 * them is reached by its name in the action panel.
 */
export interface PaletteAction {
  readonly id: string
  /** What is written on it. */
  readonly text: string
}

/**
 * One thing that can be chosen. The identifier is opaque: the palette has no
 * way to ask what it addresses, and hands it back as given.
 */
export interface PaletteItem {
  readonly id: string
  /** What it is called. */
  readonly title: string
  /** Where in the title the words stand. */
  readonly at?: readonly PaletteSpan[]
  /** A second line: where the item stands, or the words it was found among. */
  readonly detail?: string
  /** Where the words stand in that line. */
  readonly detailAt?: readonly PaletteSpan[]
  /** What can be done to it. An item offering none is drawn and not chosen. */
  readonly actions?: readonly PaletteAction[]
  /**
   * The keystroke that reaches this item away from the palette, written as the
   * caller writes it. Drawn at the end of the row.
   */
  readonly keys?: string
  /** Drawn and announced, and not choosable. */
  readonly disabled?: boolean
}

/** One band of the list. What the bands are is the caller's. */
export interface PaletteBand {
  readonly id: string
  /** What the band is called. */
  readonly title: string
  readonly items: readonly PaletteItem[]
  /** More is on its way, so what stands here is not all of it. */
  readonly working?: boolean
  /** What is said in place of items when the band holds none. */
  readonly silence?: string
}

/** One item, and the band it was drawn in. */
export interface PalettePlace {
  readonly band: PaletteBand
  readonly item: PaletteItem
}


/**
 * The bands in the order they are drawn: as they were offered, and the ones
 * holding nothing after the ones holding something.
 *
 * A band that came back with nothing is still drawn: it says the question was
 * asked and answered. It stands at the foot, and holds no item, so what the
 * keyboard counts is untouched.
 */
export const ordered = (bands: readonly PaletteBand[]): readonly PaletteBand[] => [
  ...bands.filter((one) => one.items.length > 0),
  ...bands.filter((one) => one.items.length === 0),
]

/**
 * Every item in the order it is drawn, so that one number says which item.
 */
export const flatten = (bands: readonly PaletteBand[]): readonly PalettePlace[] =>
  bands.flatMap((band) => band.items.map((item) => ({ band, item })))

/** Whether the keyboard may land here. An item with nothing to do is passed over. */
export const choosable = (item: PaletteItem): boolean =>
  !item.disabled && (item.actions?.length ?? 0) > 0

/**
 * Where the keyboard lands next, counting from `from` and passing over what
 * cannot be chosen. It wraps, and answers -1 when there is nothing to land on.
 *
 * Counting from -1 by one is how the first is asked for, and from 0 by minus
 * one is how the last is.
 */
export const stepTo = (
  places: readonly PalettePlace[],
  from: number,
  by: number,
): number => {
  const total = places.length
  for (let step = 1; step <= total; step += 1) {
    const at = (((from + by * step) % total) + total) % total
    const place = places[at]
    if (place && choosable(place.item)) return at
  }
  return -1
}

/**
 * Where the keyboard lands in a list of this many, counting from `from`. It
 * wraps, and answers -1 for a list holding nothing. Every row of such a list
 * can be landed on.
 */
export const stepIn = (total: number, from: number, by: number): number => {
  if (total <= 0) return -1
  return (((from + by) % total) + total) % total
}

/**
 * Where the keyboard stands once the list has changed under it: on the item it
 * was on, wherever that item has moved to. An item that is gone hands it to the
 * first item there is; a list with nothing to land on takes it nowhere.
 */
export const keptAt = (places: readonly PalettePlace[], was: string): number => {
  const held = places.findIndex((place) => place.item.id === was && choosable(place.item))
  return held >= 0 ? held : stepTo(places, -1, 1)
}

/**
 * The keys that reach an item's actions, in the order the actions are offered.
 * An item may offer more actions than there are keys, and those are reached by
 * name in the action panel.
 */
export const PALETTE_KEYS = ['↵', '⇧↵'] as const

/** One action, and the key that reaches it straight from the list. */
export interface PaletteKeyed {
  readonly action: PaletteAction
  readonly key: string
}

/**
 * The actions of this item a key reaches, in the order they are offered. This
 * is what the foot of the palette says, and where the keys are decided.
 */
export const keyed = (item: PaletteItem | undefined): readonly PaletteKeyed[] => {
  const actions = item && choosable(item) ? (item.actions ?? []) : []
  return actions
    .slice(0, PALETTE_KEYS.length)
    .map((action, at) => ({ action, key: PALETTE_KEYS[at] ?? '' }))
}

/** What the action Enter reaches is, and Shift and Enter the second. */
export const actionAt = (item: PaletteItem | undefined, second: boolean): string =>
  keyed(item)[second ? 1 : 0]?.action.id ?? ''

/** Whether this keystroke asks for the action panel. */
export const opensActions = (event: {
  readonly key: string
  readonly ctrlKey: boolean
  readonly metaKey: boolean
}): boolean => (event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k'

/**
 * What the key that opens the action panel is written as, from the platform a
 * browser reports. It is Command on Apple keyboards and Control everywhere
 * else.
 */
export const commandKeyWord = (platform: string): string =>
  /mac|iphone|ipad|ipod/i.test(platform) ? '⌘K' : '⌃K'

/**
 * A line split into the runs that are why the item is here and the runs that
 * are not. A line nothing stands in is one run.
 *
 * The spans are counted in UTF-16 code units, which is how the text arrives and
 * how a string is sliced. A boundary landing inside a character is moved off it,
 * outwards, so no run ends on half of one.
 */
export const partsOf = (text: string, at: readonly PaletteSpan[] = []): readonly PalettePart[] => {
  const runs = merged(text, at)
  if (runs.length === 0) return text === '' ? [] : [{ text, hit: false }]

  const out: PalettePart[] = []
  let read = 0
  for (const run of runs) {
    if (run.from > read) out.push({ text: text.slice(read, run.from), hit: false })
    out.push({ text: text.slice(run.from, run.to), hit: true })
    read = run.to
  }
  if (read < text.length) out.push({ text: text.slice(read), hit: false })
  return out
}

/**
 * The spans as runs of this text: inside it, in order, none of them empty, and
 * no two of them touching.
 */
const merged = (text: string, at: readonly PaletteSpan[]): PaletteSpan[] => {
  const kept = at
    .map((span) => ({
      from: whole(text, bounded(text, Math.min(span.from, span.to)), -1),
      to: whole(text, bounded(text, Math.max(span.from, span.to)), 1),
    }))
    .filter((span) => span.from < span.to)
    .sort((one, other) => one.from - other.from)

  const out: PaletteSpan[] = []
  for (const span of kept) {
    const last = out[out.length - 1]
    if (last && span.from <= last.to) {
      out[out.length - 1] = { from: last.from, to: Math.max(last.to, span.to) }
      continue
    }
    out.push(span)
  }
  return out
}

const bounded = (text: string, at: number): number =>
  Math.max(0, Math.min(Math.trunc(at) || 0, text.length))

/**
 * The nearest boundary that does not fall inside a character, from `at` in the
 * direction given. A character outside the basic plane is two code units, and a
 * span that names one of them names half a character.
 */
const whole = (text: string, at: number, by: 1 | -1): number => {
  let here = at
  while (here > 0 && here < text.length && inside(text, here)) here += by
  return here
}

const inside = (text: string, at: number): boolean =>
  isTrailing(text.charCodeAt(at)) && isLeading(text.charCodeAt(at - 1))

const isLeading = (unit: number): boolean => unit >= 0xd800 && unit <= 0xdbff
const isTrailing = (unit: number): boolean => unit >= 0xdc00 && unit <= 0xdfff

/** One item as it is drawn: the number it has in the list, and its lines split. */
export interface PlacedItem {
  readonly item: PaletteItem
  /** Its number in the whole list, which is what the keyboard counts in. */
  readonly at: number
  readonly name: readonly PalettePart[]
  readonly detail: readonly PalettePart[]
}

/** One band as it is drawn. */
export interface PlacedBand {
  readonly band: PaletteBand
  readonly items: readonly PlacedItem[]
}

/**
 * Every band, with its items numbered as they stand in the whole list and each
 * of their lines already split into runs.
 *
 * The bands are walked in the order they were given, which is the order the
 * keyboard counts in, so a number here is a number into `flatten`.
 */
export const placePalette = (bands: readonly PaletteBand[]): readonly PlacedBand[] => {
  let at = 0
  return bands.map((band) => ({
    band,
    items: band.items.map((item) => ({
      item,
      at: at++,
      name: partsOf(item.title, item.at),
      detail: partsOf(item.detail ?? '', item.detailAt),
    })),
  }))
}

/** One action as it is drawn in the action panel. */
export interface PlacedAction {
  readonly action: PaletteAction
  /** Its number in the panel, which is what the keyboard counts in. */
  readonly at: number
  /** Its name, with the run the words in the panel's field picked out. */
  readonly name: readonly PalettePart[]
  /** The key that reaches it without the panel, and nothing where none does. */
  readonly key: string
}

/**
 * Every action the words in the panel's field leave, numbered as it is drawn.
 * The order is the order they were offered in, which is the order the keys were
 * handed out in, so an action keeps its key wherever the words put it.
 *
 * A word is looked for anywhere in the name, and case is not part of the
 * question. Nothing typed leaves every action.
 */
export const placeActions = (
  actions: readonly PaletteAction[] = [],
  typed = '',
): readonly PlacedAction[] => {
  const word = typed.trim().toLowerCase()
  const out: PlacedAction[] = []
  for (const [offered, action] of actions.entries()) {
    const found = word === '' ? -1 : action.text.toLowerCase().indexOf(word)
    if (word !== '' && found < 0) continue
    out.push({
      action,
      at: out.length,
      name: partsOf(action.text, found < 0 ? [] : [{ from: found, to: found + word.length }]),
      key: PALETTE_KEYS[offered] ?? '',
    })
  }
  return out
}
