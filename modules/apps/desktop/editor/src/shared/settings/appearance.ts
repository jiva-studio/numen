/**
 * How the window is drawn: the theme it wears, which half of a colour pair its
 * tokens are read as, and how large it is drawn and its reading text set.
 *
 * The page is served with three style elements at the end of the head, each
 * marked as the one of the three it is: the mode, the theme's own file, and the
 * two multipliers. The theme's stands after the mode's and the sizes after both.
 *
 * A theme and a mode are worn the moment the keyboard lands on them. A size is
 * held until the keyboard has stood on the row for HELD, and what the settings
 * name is put back the moment the keyboard leaves the list.
 */
import { computed, ref, shallowRef, watch } from 'vue'
import { answerGuard } from '../questions'
import type { StepGroup, StepRow } from '../command/lists'
import { following, percent } from '@numen/ui'
import type { MessageWriter } from '../notices/messages'
import type { Appearance, Bounds, Mode, Ranges, Scales, Sizes, Theme, Themes } from './theme'

/** Everything the appearance says in the window's voice. */
export interface Words {
  /** The two shelves the themes are drawn in, and where a person's own go. */
  readonly shipping: string
  readonly owned: string
  readonly noneOwned: string
  /** The group the three modes are drawn in. */
  readonly half: string
  /** The row a list of values opens on, which is the value in force. */
  readonly current: string
  /** The three modes, by the name each is offered under. */
  readonly system: string
  readonly light: string
  readonly dark: string
  /** Why a mode cannot be chosen: the theme worn declares light and dark itself. */
  readonly pinned: string
  /** The group each of the two sizes is drawn in. */
  readonly drawing: string
  readonly setting: string
  /** The themes could not be listed, and one theme's file could not be read. */
  readonly unlisted: string
  readonly unworn: string
}

/** The command whose step offers the themes, and the one that offers the modes. */
export const APPEARANCE = 'appearance'
export const MODE = 'mode'

/** The command that offers how large the interface is drawn, and how large the text is set. */
export const INTERFACE_SCALE = 'interfaceScale'
export const TEXT_SCALE = 'textScale'

/** The commands whose step offers a list this holds. */
export const DRESSING: readonly string[] = [APPEARANCE, MODE, INTERFACE_SCALE, TEXT_SCALE]

/** Which of the two sizes a row is one of. */
type ScaleKind = typeof INTERFACE_SCALE | typeof TEXT_SCALE

/** One size, and which of the two it is. */
interface ScaleChoice {
  readonly which: ScaleKind
  readonly size: number
}

/** The multiplier that draws everything the size it was designed at. */
const DESIGNED = 1

/** How many sizes stand between one whole and the next. */
const STEPS = 10

/**
 * A number a person typed, which is a whole number of percent. The row it makes
 * is titled in the digits that were typed, so the words typed always leave it
 * standing.
 */
const TYPED = /^(\d+)\s*%?$/

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
const sizing = (which: ScaleKind, size: number): string => `${which}:${size}`

/** The size a row names, and nothing for a row naming anything else. */
const sizeOf = (item: string): ScaleChoice | null => {
  const [which, said] = item.split(':')
  if (which !== INTERFACE_SCALE && which !== TEXT_SCALE) return null
  const size = Number(said)
  return size > 0 ? { which, size } : null
}

/** The elements the page carries: the mode's, the theme's, and the sizes'. */
interface StyleElements {
  readonly mode: HTMLStyleElement
  readonly theme: HTMLStyleElement
  /** Nothing for a page served at no size of its own, until one is written. */
  sizes: HTMLStyleElement | undefined
}

/**
 * The attribute each of the three carries, and what each of them says it is.
 * The application writes these where it dresses the page.
 */
export const MARKER = 'data-appearance'
export const IS_MODE = 'mode'
export const IS_THEME = 'theme'
export const IS_SIZES = 'sizes'

/** The element the page was served marked as one of the three, and nothing where it carries none. */
const marked = (sheet: Document, is: string): HTMLStyleElement | null =>
  sheet.head.querySelector<HTMLStyleElement>(`style[${MARKER}="${is}"]`)

/**
 * The elements the head ends with. A page served by something that dresses it
 * in nothing is given a mode's and a theme's of its own, in that order.
 */
const dressing = (sheet: Document): StyleElements => {
  const mode = marked(sheet, IS_MODE) ?? sheet.head.appendChild(styling(IS_MODE, sheet))
  const theme = marked(sheet, IS_THEME) ?? after(mode, IS_THEME, sheet)
  return { mode, theme, sizes: marked(sheet, IS_SIZES) ?? undefined }
}

/** One of the three, marked as which of them it is. */
const styling = (is: string, sheet: Document): HTMLStyleElement => {
  const one = sheet.createElement('style')
  one.setAttribute(MARKER, is)
  return one
}

/** An element straight after another, which is where the next of the three goes. */
const after = (before: HTMLStyleElement, is: string, sheet: Document): HTMLStyleElement => {
  const next = styling(is, sheet)
  before.after(next)
  return next
}

/** The two multipliers as the page carries them. */
const declared = (sizes: Sizes): string =>
  `:root { --numen-interface-scale: ${sizes.interfaceScale}; --numen-text-scale: ${sizes.textScale}; }`

/** Whether a range reaches a size. A range holding nothing reaches none. */
const reaches = (range: Bounds, size: number): boolean =>
  range.most > range.least && range.least > 0 && size >= range.least && size <= range.most

/**
 * The sizes standing between the ends of a range: every step inside it, with
 * each end itself, so the end is offered wherever it falls.
 */
const ladder = (range: Bounds): readonly number[] => {
  if (!reaches(range, range.least)) return []
  const rungs = [range.least]
  const first = Math.ceil(range.least * STEPS)
  const last = Math.floor(range.most * STEPS)
  for (let step = first; step <= last; step += 1) {
    const size = step / STEPS
    if (size > range.least && size < range.most) rungs.push(size)
  }
  return [...rungs, range.most]
}

/** The size those digits name, and nothing where what was typed is not digits. */
const typedSize = (typed: string): number | null => {
  const said = TYPED.exec(typed.trim())
  return said ? Number(said[1]) / 100 : null
}

/**
 * The rows one to a title. A size the list already holds is not held twice, and
 * the first of a pair is the one that stands.
 */
const once = (rows: readonly StepRow[]): readonly StepRow[] => {
  const seen = new Set<string>()
  const only: StepRow[] = []
  for (const one of rows) {
    if (seen.has(one.title)) continue
    seen.add(one.title)
    only.push(one)
  }
  return only
}

/** A range until the application has said what one is. */
const NOWHERE: Bounds = { least: 0, most: 0 }

/** What is said of one of the two sizes. */
const its = <T,>(both: Scales<T>, which: ScaleKind): T => both[which]

/** The pair with what is said of one of the two put in its place. */
const onto = <T,>(both: Scales<T>, which: ScaleKind, one: T): Scales<T> => ({ ...both, [which]: one })

export function windowAppearance(
  core: Themes,
  words: Words,
  said: MessageWriter,
  sheet: Document = document,
  wait: (ms: number) => Promise<unknown> = sleep,
) {
  const dressed = dressing(sheet)
  /** What the page was served wearing, which is the applied theme's file. */
  const served = dressed.theme.textContent ?? ''
  /** What the sizes' element holds, once the window knows what it was served at. */
  let written = ''

  /** Every theme there is, and the theme and the mode the settings name. */
  const list = shallowRef<readonly Theme[]>([])
  const applied = ref('')
  const mode = ref<Mode>('system')

  /** The two sizes the settings name, and how far each of them goes. */
  const settings = ref<Sizes>({ interfaceScale: DESIGNED, textScale: DESIGNED })
  const bounds = ref<Ranges>({ interfaceScale: NOWHERE, textScale: NOWHERE })

  /**
   * What the appearance lost touch with, said until it has it back.
   *
   * It is a state and not a word, so the window carries it where it carries
   * everything else that is simply so.
   */
  const lost = ref('')

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
  const holding = ref<ScaleChoice | null>(null)
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
  const isPinned = computed(() => {
    const theme = list.value.find((one) => one.name === worn.value)
    return theme?.isPinned ?? theme?.pinned ?? false
  })
  const pinned = isPinned

  /** A file arriving for a row the keyboard has already left is dropped. */
  const asks = answerGuard()

  let open = true
  /** Let go of the stream the window is listening to. */
  const listening = new AbortController()
  const follows = following({
    open: () => open,
    lost: (gone) => (lost.value = gone),
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
    const mine = asks.ask()
    dressed.mode.textContent = `:root { color-scheme: ${SCHEMES[half.value]}; }`
    try {
      const css = await fileOf(worn.value)
      if (mine.current) dressed.theme.textContent = css
    } catch (error) {
      // The reason goes to the console; the person is told in the window's
      // own voice.
      console.error(error)
      if (mine.current) said(words.unworn, 'refusal')
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
    dressed.sizes ??= after(dressed.theme, IS_SIZES, sheet)
    dressed.sizes.textContent = css
  }

  /** The window drawn again wherever the size it is drawn at changes. */
  const drawing = watch(sized, draws)

  /** Every theme there is, and what the settings say the window is drawn as. */
  const lists = async () => {
    let answer: Appearance
    try {
      answer = await core.appearance()
    } catch (error) {
      console.error(error)
      said(words.unlisted, 'refusal')
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
   * came off is said by the group it stands in, so a row says only what is
   * true of it alone.
   */
  const shelf = (shipping: boolean): readonly StepRow[] => {
    const off = list.value.filter((one) => one.shipped === shipping)
    const named = (one: Theme): StepRow => ({
      id: one.name,
      title: one.title,
      ...(one.name === applied.value ? { detail: words.current, inForce: true } : {}),
    })
    return [
      ...off.filter((one) => one.name === applied.value).map(named),
      ...off.filter((one) => one.name !== applied.value).map(named),
    ]
  }

  /**
   * The themes, in the two groups they come off. The group the theme worn came
   * off stands first, so opening the list stands on what the window wears.
   */
  const offers = (): readonly StepGroup[] => {
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
    return one === mode.value ? words.current : ''
  }

  /**
   * The three modes. Each is drawn as not to be chosen while the theme worn
   * pins light and dark: it is there, it says why, and the keyboard passes
   * over it.
   */
  const modes = (): readonly StepGroup[] => {
    const row = (one: Mode): StepRow => {
      const detail = beside(one)
      return {
        id: named(one),
        title: words[one],
        ...(detail ? { detail } : {}),
        ...(one === mode.value ? { inForce: true } : {}),
        ...(pinned.value ? { disabled: true } : {}),
      }
    }
    return [{ id: 'half', title: words.half, items: MODES.map(row) }]
  }

  /**
   * The sizes one of the two commands offers, in one group of its own: every
   * step the range reaches, the size the window is drawn at, and the number a
   * person typed. Each stands once, in order, and the range is what a size has
   * to be inside to stand at all.
   *
   * The one row that says anything is the size the window is drawn at, which is
   * where a person is standing before they walk.
   */
  const sizes = (command: string, typed = ''): readonly StepGroup[] => {
    const which: ScaleKind = command === TEXT_SCALE ? TEXT_SCALE : INTERFACE_SCALE
    const range = its(bounds.value, which)
    const now = its(settings.value, which)
    const said = typedSize(typed)
    const row = (size: number): StepRow => ({
      id: sizing(which, size),
      title: percent(size),
      ...(size === now ? { detail: words.current, inForce: true } : {}),
    })
    const held = [...ladder(range), now, ...(said === null ? [] : [said])]
      .filter((size) => reaches(range, size))
      .sort((first, second) => first - second)
    const title = which === INTERFACE_SCALE ? words.drawing : words.setting
    return [{ id: which, title, items: once(held.map(row)) }]
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
    if (size && reaches(its(bounds.value, size.which), size.size)) {
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
    said('')
    applied.value = chosen ? was.applied : item
    mode.value = chosen ?? was.mode
    stood.value = ''
    await puts()

    const failed = await core.chooses(applied.value, mode.value, settings.value)
    if (!failed) return
    said(failed, 'refusal')
    applied.value = was.applied
    mode.value = was.mode
    await puts()
  }

  /**
   * The size chosen: the window is drawn at it at once, whatever the hold was
   * waiting for, and it is written into the settings beside the theme. A number
   * the settings refuse is said, and the window goes back to the size they hold.
   */
  const picks = async (chosen: ScaleChoice) => {
    if (!reaches(its(bounds.value, chosen.which), chosen.size)) return
    const was = settings.value
    said('')
    clearTimeout(holds)
    holding.value = null
    stood.value = ''
    settings.value = onto(was, chosen.which, chosen.size)

    const failed = await core.chooses(applied.value, mode.value, settings.value)
    if (!failed) return
    said(failed, 'refusal')
    settings.value = was
  }

  return {
    list,
    applied,
    mode,
    sized,
    bounds,
    isPinned,
    pinned,
    lost,
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
