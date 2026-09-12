/**
 * The window as it is drawn — the theme it wears, which half of a colour pair
 * its tokens are read as, and how large it is drawn and its reading text set.
 *
 * A theme and a mode are worn the moment the keyboard lands on them. A size is
 * held until the keyboard has stood on the row for HELD, and what the settings
 * name is put back the moment the keyboard leaves the list.
 */
import { ref, shallowRef, watch } from 'vue'
import { createFollower } from '@numen/ui'
import { answerGuard } from '@/shared/questions'
import type { MessageWriter } from '@/shared/notices/messages'
import { DESIGNED, NOWHERE, SCHEMES } from '@/entities/settings'
import type { Appearance, Mode, Ranges, Sizes, Theme, Themes } from '@/entities/settings'
import type { AppearanceWords } from '../words'
import { IS_SIZES, after, getSizesCss, getStyleElements } from '../lib/head'
import { getModeGroups, getSizeGroups, getThemeGroups } from '../lib/offers'
import { useAppearanceChoice } from './choice'

export function windowAppearance(
  core: Themes,
  words: AppearanceWords,
  write: MessageWriter,
  sheet: Document = document,
  wait: (ms: number) => Promise<unknown> = sleep,
) {
  const dressed = getStyleElements(sheet)
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

  /** A file arriving for a row the keyboard has already left is dropped. */
  const asks = answerGuard()

  /** One theme's file, read once and kept. */
  const fileOf = async (name: string): Promise<string> => {
    const kept = files.get(name)
    if (kept !== undefined) return kept
    const css = await core.readTheme(name)
    files.set(name, css)
    return css
  }

  /** The theme and the mode named, written into the two elements the page was served with. */
  const wear = async (theme: string, half: Mode) => {
    // The page arrived dressed, and nothing is written over that until the
    // window has been told what it is dressed in.
    if (!applied.value) return
    const mine = asks.ask()
    dressed.mode.textContent = `:root { color-scheme: ${SCHEMES[half]}; }`
    try {
      const css = await fileOf(theme)
      if (mine.current) dressed.theme.textContent = css
    } catch (error) {
      // The reason goes to the console; the person is told in the window's
      // own voice.
      console.error(error)
      if (mine.current) write(words.unworn, 'error')
    }
  }

  /** The row the keyboard stands on, and what standing on it or choosing it does. */
  const choice = useAppearanceChoice({
    core,
    write,
    list,
    applied,
    mode,
    settings,
    bounds,
    wear,
  })

  /** What the window wears now, which is the row stood on over the settings. */
  const applyAppearance = () => wear(choice.worn.value, choice.half.value)

  /**
   * The two multipliers, written into the element the head ends with. The page
   * was served carrying them, so what already stands there is left alone.
   */
  const writeSizes = () => {
    const css = getSizesCss(choice.sized.value)
    if (css === written) return
    written = css
    dressed.sizes ??= after(dressed.theme, IS_SIZES, sheet)
    dressed.sizes.textContent = css
  }

  /** The window drawn again wherever the size it is drawn at changes. */
  const drawing = watch(choice.sized, writeSizes)

  let open = true
  /** Let go of the stream the window is listening to. */
  const listening = new AbortController()
  const follows = createFollower({
    isOpen: () => open,
    setLost: (gone) => (lost.value = gone),
    wait,
  })

  /** Every theme there is, and what the settings say the window is drawn as. */
  const loadAppearance = async () => {
    let answer: Appearance
    try {
      answer = await core.getAppearance()
    } catch (error) {
      console.error(error)
      write(words.unlisted, 'error')
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
      written = dressed.sizes?.textContent ?? getSizesCss(answer.sizes)
    }
    arrived = false
  }

  /**
   * The person's folder changed. What changed is read again from the file it
   * is now, and a theme being worn is worn as it now reads.
   */
  const follow = () =>
    follows(
      () => core.watchThemes(listening.signal),
      async (names) => {
        if (!names.length) return
        for (const name of names) files.delete(name)
        await loadAppearance()
        if (names.includes(choice.worn.value)) await applyAppearance()
      },
    )

  /**
   * The list, and the folder followed. Nothing is worn from here: the page
   * arrived dressed, and this is what it was dressed in.
   */
  const start = async () => {
    await loadAppearance()
    void follow()
  }

  const close = () => {
    open = false
    listening.abort()
    choice.close()
    drawing()
  }

  const listThemeGroups = () => getThemeGroups(list.value, applied.value, words)
  const listModeGroups = () => getModeGroups(mode.value, choice.isPinned.value, words)
  const listSizeGroups = (command: string, text = '') =>
    getSizeGroups(command, text, settings.value, bounds.value, words)

  return {
    list,
    applied,
    mode,
    sized: choice.sized,
    bounds,
    isPinned: choice.isPinned,
    pinned: choice.isPinned,
    lost,
    getThemeGroups: listThemeGroups,
    getModeGroups: listModeGroups,
    getSizeGroups: listSizeGroups,
    previewItem: choice.previewItem,
    chooseItem: choice.chooseItem,
    start,
    close,
  }
}

const sleep = (ms: number) => new Promise((wake) => setTimeout(wake, ms))
