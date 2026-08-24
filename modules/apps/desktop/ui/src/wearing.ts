/**
 * The theme the window wears, and the list of the ones it could.
 *
 * The page is served already wearing one, as the two style elements the head
 * ends with: which half of a colour pair the tokens are read as, and the
 * theme's own file. What is written here is what those two hold. Neither of
 * them moves, because the theme's has to stand after the mode's.
 *
 * A theme is worn the moment the keyboard lands on it, and the one the
 * settings name is put back the moment the keyboard leaves the list.
 */
import { computed, ref, shallowRef } from 'vue'
import type { Offered, Offering } from './commanding'
import { following } from './following'
import type { Catalogue, Mode, Themes, Wearable } from './theme'

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
  /** The themes could not be listed, and one theme's file could not be read. */
  readonly unlisted: string
  readonly unworn: string
}

/** The command whose step offers the themes, and the one that offers the modes. */
export const APPEARANCE = 'appearance'
export const MODE = 'mode'

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

/** The two elements the page carries: the mode's, and the theme's after it. */
interface Dressed {
  readonly mode: HTMLStyleElement
  readonly theme: HTMLStyleElement
}

/** The declaration that says which half of a pair every token is read as. */
const SCHEME = /^\s*:root\s*\{\s*color-scheme:/

/**
 * The two elements the head ends with. A page served by something that dresses
 * it in nothing is given a pair of its own, in that order.
 */
const dressing = (sheet: Document): Dressed => {
  const styles = [...sheet.head.querySelectorAll('style')]
  const at = styles.reduce(
    (last, one, index) => (SCHEME.test(one.textContent ?? '') ? index : last),
    -1,
  )
  const mode = styles[at] ?? sheet.head.appendChild(sheet.createElement('style'))
  const theme = (at < 0 ? undefined : styles[at + 1]) ?? after(mode, sheet)
  return { mode, theme }
}

/** A second element, straight after the mode's, which is where a theme goes. */
const after = (mode: HTMLStyleElement, sheet: Document): HTMLStyleElement => {
  const theme = sheet.createElement('style')
  mode.after(theme)
  return theme
}

export function wearing(
  core: Themes,
  words: Words,
  sheet: Document = document,
  wait: (ms: number) => Promise<unknown> = sleep,
) {
  const dressed = dressing(sheet)
  /** What the page was served wearing, which is the applied theme's file. */
  const served = dressed.theme.textContent ?? ''

  /** Every theme there is, and the theme and the mode the settings name. */
  const list = shallowRef<readonly Wearable[]>([])
  const applied = ref('')
  const mode = ref<Mode>('system')

  /** What could not be listed, read or written, in words a person reads. */
  const said = ref('')

  /** The text of every theme that has been worn, by name. */
  const files = new Map<string, string>()

  /** Whether what the page arrived in still stands for the theme applied. */
  let arrived = true

  /** The row the keyboard is standing on, and nothing while the list is shut. */
  const stood = ref('')

  /** The theme worn now: the one the keyboard is on, else the one applied. */
  const worn = computed(() =>
    stood.value && !modeOf(stood.value) ? stood.value : applied.value,
  )

  /** The mode read now, the same way. */
  const half = computed<Mode>(() => modeOf(stood.value) ?? mode.value)

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

  /** Every theme there is, and which of them the settings name. */
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
    // The page was served wearing this one, so its file has been read already.
    if (arrived) files.set(answer.applied, served)
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

  /**
   * The row the keyboard is standing on, worn while it stands there. Nothing
   * standing puts back the theme and the mode the settings name.
   */
  const shows = (item: string) => {
    stood.value = item
    void puts()
  }

  /**
   * The row chosen: it is worn at once and written into the settings. Settings
   * that could not be written say so, and the window wears what they hold.
   */
  const chooses = async (item: string) => {
    const was = { applied: applied.value, mode: mode.value }
    const chosen = modeOf(item)
    if (!chosen && !list.value.some((one) => one.name === item)) return
    if (chosen && pinned.value) return
    said.value = ''
    applied.value = chosen ? was.applied : item
    mode.value = chosen ?? was.mode
    stood.value = ''
    await puts()

    const failed = await core.chooses(applied.value, mode.value)
    if (!failed) return
    said.value = failed
    applied.value = was.applied
    mode.value = was.mode
    await puts()
  }

  return { list, applied, mode, said, offers, modes, shows, chooses, start, close }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))
