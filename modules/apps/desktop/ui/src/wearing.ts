/**
 * How the window is drawn: the theme it wears, which half of a colour pair its
 * tokens are read as, and how large it is drawn and its reading text set.
 *
 * The page is served already drawn that way, as the three style elements the
 * head ends with: the mode, the theme's own file, and the two multipliers.
 * What is written here is what those three hold. None of them moves, because
 * the theme's has to stand after the mode's and the sizes after both.
 *
 * A theme and a mode are worn the moment the keyboard lands on them. A size is
 * held: the keyboard has to have stood on the row for HELD before the window
 * is drawn at it, because a size relays out every document that is open. What
 * the settings name is put back the moment the keyboard leaves the list.
 */
import { computed, ref, shallowRef, watch } from 'vue'
import type { Offered, Offering } from './commanding'
import { following } from './following'
import type { Both, Bounds, Catalogue, Mode, Ranges, Sizes, Themes, Wearable } from './theme'

/** Everything the appearance says in the window's voice. */
export interface Words {
  /** The two shelves the themes are drawn in, and where a person's own go. */
  readonly shipping: string
  readonly owned: string
  readonly noneOwned: string
  /** The band the three modes are drawn in. */
  readonly half: string
  /** The theme and the mode the settings name, said on their rows. */
  readonly worn: string
  /** The three modes, by the name each is offered under. */
  readonly system: string
  readonly light: string
  readonly dark: string
  /** Why a mode cannot be chosen: the theme worn declares light and dark itself. */
  readonly pinned: string
  /** The band each of the two sizes is drawn in. */
  readonly drawing: string
  readonly setting: string
  /** The size the settings name, and the size everything was designed at. */
  readonly sized: string
  readonly designed: string
  /** The themes could not be listed, and one theme's file could not be read. */
  readonly unlisted: string
  readonly unworn: string
}

/** The command whose step offers the themes, and the one that offers the modes. */
export const APPEARANCE = 'appearance'
export const MODE = 'mode'

/** The command that offers how large the interface is drawn, and how large the text is set. */
export const INTERFACE = 'interface'
export const FONT = 'font'

/** The commands whose step offers a list this holds. */
export const DRESSING: readonly string[] = [APPEARANCE, MODE, INTERFACE, FONT]

/** Which of the two sizes a row is one of. */
type Which = typeof INTERFACE | typeof FONT

/** One size, and which of the two it is. */
interface Sized {
  readonly which: Which
  readonly size: number
}

/** The multiplier that draws everything the size it was designed at. */
const DESIGNED = 1

/** How far apart the sizes a person is offered stand. */
const APART = 0.25

/**
 * How long the keyboard has to have stood on a size before the window is drawn
 * at it. A held arrow key crosses a row every 40 milliseconds, and each row
 * applied is every open document laid out again and its pages emptied.
 */
const HELD = 150

/** The modes, in the order they are offered. */
const MODES: readonly Mode[] = ['system', 'light', 'dark']

/**
 * What a mode is offered under: the command it belongs to, and the mode. A
 * theme is named by its shelf, and there are two shelves, so no theme is ever
 * named this.
 */
const named = (one: Mode): string => `${MODE}:${one}`

/** What `color-scheme` is written as for each mode. */
const SCHEMES: Record<Mode, string> = {
  system: 'light dark',
  light: 'light',
  dark: 'dark',
}

/** The mode a row names, and nothing for a row naming a theme. */
const modeOf = (item: string): Mode | null =>
  MODES.find((one) => named(one) === item) ?? null

/**
 * What a size is offered under: the command it belongs to, and the multiplier.
 * A theme is named by its shelf, and there are two shelves, so no theme is
 * ever named this.
 */
const sizing = (which: Which, size: number): string => `${which}:${size}`

/** The size a row names, and nothing for a row naming anything else. */
const sizeOf = (item: string): Sized | null => {
  const [which, said] = item.split(':')
  if (which !== INTERFACE && which !== FONT) return null
  const size = Number(said)
  return size > 0 ? { which, size } : null
}

/** The elements the page carries: the mode's, the theme's, and the sizes'. */
interface Dressed {
  readonly mode: HTMLStyleElement
  readonly theme: HTMLStyleElement
  /** Nothing for a page served at no size of its own, until one is written. */
  sizes: HTMLStyleElement | undefined
}

/** The declaration that says which half of a pair every token is read as. */
const SCHEME = /^\s*:root\s*\{\s*color-scheme:/

/** The declaration the two multipliers stand in. */
const SIZED = /--numen-(?:interface|font)\s*:/

/**
 * The elements the head ends with. A page served by something that dresses it
 * in nothing is given a mode's and a theme's of its own, in that order.
 */
const dressing = (sheet: Document): Dressed => {
  const styles = [...sheet.head.querySelectorAll('style')]
  const at = styles.reduce(
    (last, one, index) => (SCHEME.test(one.textContent ?? '') ? index : last),
    -1,
  )
  const mode = styles[at] ?? sheet.head.appendChild(sheet.createElement('style'))
  const theme = (at < 0 ? undefined : styles[at + 1]) ?? after(mode, sheet)
  // Looked for after the theme's, which is the one place it stands. A theme's
  // own file may declare either multiplier.
  const sizes =
    at < 0 ? undefined : styles.slice(at + 2).find((one) => SIZED.test(one.textContent ?? ''))
  return { mode, theme, sizes }
}

/** An element straight after another, which is where the next of the three goes. */
const after = (before: HTMLStyleElement, sheet: Document): HTMLStyleElement => {
  const next = sheet.createElement('style')
  before.after(next)
  return next
}

/** The two multipliers as the page carries them. */
const declared = (sizes: Sizes): string =>
  `:root { --numen-interface: ${sizes.interface}; --numen-font: ${sizes.font}; }`

/**
 * The sizes offered between the ends of a range: every multiple of APART
 * inside it, with each end itself. A range holding nothing offers nothing.
 */
const ladder = (bounds: Bounds): readonly number[] => {
  if (!(bounds.most > bounds.least) || bounds.least <= 0) return []
  const rungs = [bounds.least]
  const first = Math.ceil(bounds.least / APART)
  const last = Math.floor(bounds.most / APART)
  for (let step = first; step <= last; step += 1) {
    const size = step * APART
    if (size > bounds.least && size < bounds.most) rungs.push(size)
  }
  return [...rungs, bounds.most]
}

/** A multiplier as a person reads it. */
const percent = (size: number): string => `${Math.round(size * 100)}%`

/** A range until the application has said what one is. */
const NOWHERE: Bounds = { least: 0, most: 0 }

/** What is said of one of the two sizes. */
const its = <T,>(both: Both<T>, which: Which): T =>
  which === INTERFACE ? both.interface : both.font

/** The pair with what is said of one of the two put in its place. */
const onto = <T,>(both: Both<T>, which: Which, one: T): Both<T> =>
  which === INTERFACE ? { ...both, interface: one } : { ...both, font: one }

export function wearing(
  core: Themes,
  words: Words,
  sheet: Document = document,
  wait: (ms: number) => Promise<unknown> = sleep,
) {
  const dressed = dressing(sheet)
  /** What the page was served wearing, which is the applied theme's file. */
  const served = dressed.theme.textContent ?? ''
  /** What the sizes' element holds, once the window knows what it was served at. */
  let written = ''

  /** Every theme there is, and the theme and the mode the settings name. */
  const list = shallowRef<readonly Wearable[]>([])
  const applied = ref('')
  const mode = ref<Mode>('system')

  /** The two sizes the settings name, and how far each of them goes. */
  const settings = ref<Sizes>({ interface: DESIGNED, font: DESIGNED })
  const bounds = ref<Ranges>({ interface: NOWHERE, font: NOWHERE })

  /** What could not be listed, read or written, in words a person reads. */
  const said = ref('')

  /** The text of every theme that has been worn, by name. */
  const files = new Map<string, string>()

  /** Whether what the page arrived in still stands for the theme applied. */
  let arrived = true

  /** The row the keyboard is standing on, and nothing while the list is shut. */
  const stood = ref('')

  /**
   * The size the keyboard has stood on long enough for the window to be drawn
   * at it, and nothing while it stands anywhere else.
   */
  const holding = ref<Sized | null>(null)
  let holds: ReturnType<typeof setTimeout> | undefined

  /** Whether a row names a theme, which the rows of the other lists do not. */
  const themed = (item: string): boolean => item !== '' && !modeOf(item) && !sizeOf(item)

  /** The theme worn now: the one the keyboard is on, else the one applied. */
  const worn = computed(() => (themed(stood.value) ? stood.value : applied.value))

  /** The mode read now, the same way. */
  const half = computed<Mode>(() => modeOf(stood.value) ?? mode.value)

  /** The sizes the window is drawn at now: the one being held, over the settings'. */
  const sized = computed<Sizes>(() => {
    const held = holding.value
    return held ? onto(settings.value, held.which, held.size) : settings.value
  })

  /** Whether the theme worn declares light and dark itself. */
  const pinned = computed(
    () => list.value.find((one) => one.name === worn.value)?.pinned ?? false,
  )

  /**
   * Which dressing is the current one. A file arriving for a row the keyboard
   * has already left is dropped.
   */
  let asked = 0

  let open = true
  /** Let go of the stream the window is listening to. */
  const listening = new AbortController()
  const follows = following({
    open: () => open,
    lost: (lost) => {
      said.value = lost
    },
    wait,
  })

  /** One theme's file, read once and kept. */
  const fileOf = async (name: string): Promise<string> => {
    const kept = files.get(name)
    if (kept !== undefined) return kept
    const css = await core.text(name)
    files.set(name, css)
    return css
  }

  /** What the window wears now, written into the two elements it was served with. */
  const puts = async () => {
    // The page arrived dressed, and nothing is written over that until the
    // window has been told what it is dressed in.
    if (!applied.value) return
    const mine = ++asked
    dressed.mode.textContent = `:root { color-scheme: ${SCHEMES[half.value]}; }`
    try {
      const css = await fileOf(worn.value)
      if (mine === asked) dressed.theme.textContent = css
    } catch (error) {
      // The reason goes to the console; the person is told in the window's
      // own voice.
      console.error(error)
      if (mine === asked) said.value = words.unworn
    }
  }

  /**
   * The two multipliers, written into the element the head ends with. The page
   * was served carrying them, so what already stands there is left alone.
   */
  const draws = () => {
    const css = declared(sized.value)
    if (css === written) return
    written = css
    dressed.sizes ??= after(dressed.theme, sheet)
    dressed.sizes.textContent = css
  }

  /** The window drawn again wherever the size it is drawn at changes. */
  const drawing = watch(sized, draws)

  /** Every theme there is, and what the settings say the window is drawn as. */
  const lists = async () => {
    let answer: Catalogue
    try {
      answer = await core.catalogue()
    } catch (error) {
      console.error(error)
      said.value = words.unlisted
      return
    }
    list.value = answer.themes
    applied.value = answer.applied
    mode.value = answer.mode
    settings.value = answer.sizes
    bounds.value = answer.bounds
    if (arrived) {
      // The page was served wearing this theme, so its file has been read
      // already, and drawn at these sizes, so they already stand in the head.
      files.set(answer.applied, served)
      written = dressed.sizes?.textContent ?? declared(answer.sizes)
    }
    arrived = false
  }

  /**
   * The person's folder changed. What changed is read again from the file it
   * is now, and a theme being worn is worn as it now reads.
   */
  const follow = () =>
    follows(
      () => core.changed(listening.signal),
      async (names) => {
        if (!names.length) return
        for (const name of names) files.delete(name)
        await lists()
        if (names.includes(worn.value)) await puts()
      },
    )

  /**
   * The list, and the folder followed. Nothing is worn from here: the page
   * arrived dressed, and this is what it was dressed in.
   */
  const start = async () => {
    await lists()
    void follow()
  }

  const close = () => {
    open = false
    listening.abort()
    clearTimeout(holds)
    drawing()
  }

  /**
   * The themes off one shelf, the one the settings name first. Where a theme
   * came off is said by the band it stands in, so a row says only what is
   * true of it alone.
   */
  const shelf = (shipping: boolean): readonly Offered[] => {
    const off = list.value.filter((one) => one.shipped === shipping)
    const named = (one: Wearable): Offered => ({
      id: one.name,
      title: one.title,
      ...(one.name === applied.value ? { detail: words.worn } : {}),
    })
    return [
      ...off.filter((one) => one.name === applied.value).map(named),
      ...off.filter((one) => one.name !== applied.value).map(named),
    ]
  }

  /**
   * The themes, in the two bands they come off. The band the theme worn came
   * off stands first, so opening the list stands on what the window wears.
   */
  const offers = (): readonly Offering[] => {
    const shipping = { id: 'shipping', title: words.shipping, items: shelf(true) }
    const own = {
      id: 'owned',
      title: words.owned,
      items: shelf(false),
      silence: words.noneOwned,
    }
    const worn = list.value.find((one) => one.name === applied.value)
    return worn && !worn.shipped ? [own, shipping] : [shipping, own]
  }

  /** What is said about a mode: why it cannot be chosen, or that it is the one. */
  const beside = (one: Mode): string => {
    if (pinned.value) return words.pinned
    return one === mode.value ? words.worn : ''
  }

  /**
   * The three modes. Each is drawn as not to be chosen while the theme worn
   * pins light and dark: it is there, it says why, and the keyboard passes
   * over it.
   */
  const modes = (): readonly Offering[] => {
    const row = (one: Mode): Offered => {
      const detail = beside(one)
      return {
        id: named(one),
        title: words[one],
        ...(detail ? { detail } : {}),
        ...(pinned.value ? { disabled: true } : {}),
      }
    }
    return [{ id: 'half', title: words.half, items: MODES.map(row) }]
  }

  /** What is said beside a size: that it is the one, or that it is as designed. */
  const alongside = (which: Which, size: number): string => {
    if (size === its(settings.value, which)) return words.sized
    return size === DESIGNED ? words.designed : ''
  }

  /**
   * The sizes one of the two commands offers, in one band of its own. A range
   * the application has not answered with offers nothing.
   */
  const sizes = (command: string): readonly Offering[] => {
    const which: Which = command === FONT ? FONT : INTERFACE
    const row = (size: number): Offered => {
      const detail = alongside(which, size)
      return {
        id: sizing(which, size),
        title: percent(size),
        ...(detail ? { detail } : {}),
      }
    }
    const title = which === INTERFACE ? words.drawing : words.setting
    return [{ id: which, title, items: ladder(its(bounds.value, which)).map(row) }]
  }

  /**
   * The row the keyboard is standing on, worn while it stands there. A size is
   * drawn only once the keyboard has stood on it for HELD. Nothing standing
   * puts back what the settings name.
   */
  const shows = (item: string) => {
    stood.value = item
    clearTimeout(holds)
    const size = sizeOf(item)
    if (size) {
      holds = setTimeout(() => (holding.value = size), HELD)
      return
    }
    holding.value = null
    void puts()
  }

  /**
   * The row chosen: it is worn at once and written into the settings. Settings
   * that could not be written say so, and the window wears what they hold.
   */
  const chooses = async (item: string) => {
    const size = sizeOf(item)
    if (size) return await picks(size)
    const was = { applied: applied.value, mode: mode.value }
    const chosen = modeOf(item)
    if (!chosen && !list.value.some((one) => one.name === item)) return
    if (chosen && pinned.value) return
    said.value = ''
    applied.value = chosen ? was.applied : item
    mode.value = chosen ?? was.mode
    stood.value = ''
    await puts()

    const failed = await core.chooses(applied.value, mode.value, settings.value)
    if (!failed) return
    said.value = failed
    applied.value = was.applied
    mode.value = was.mode
    await puts()
  }

  /**
   * The size chosen: the window is drawn at it at once, whatever the hold was
   * waiting for, and it is written into the settings beside the theme. A number
   * the settings refuse is said, and the window goes back to the size they hold.
   */
  const picks = async (chosen: Sized) => {
    if (!ladder(its(bounds.value, chosen.which)).includes(chosen.size)) return
    const was = settings.value
    said.value = ''
    clearTimeout(holds)
    holding.value = null
    stood.value = ''
    settings.value = onto(was, chosen.which, chosen.size)

    const failed = await core.chooses(applied.value, mode.value, settings.value)
    if (!failed) return
    said.value = failed
    settings.value = was
  }

  return {
    list,
    applied,
    mode,
    sized,
    said,
    offers,
    modes,
    sizes,
    shows,
    chooses,
    start,
    close,
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))
