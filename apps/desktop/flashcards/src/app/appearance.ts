/**
 * The appearance commands of the window: the themes the application lists and the three modes of
 * a color pair, offered as items of the shell's command palette. Choosing one writes it to the
 * settings through the theme service and dresses the page in it.
 */
import { ref, shallowRef, type Ref, type ShallowRef } from 'vue'
import {
  THEME_MODE,
  THEME_ORIGIN,
  formatErrorMessage,
  type ThemeModeCode,
  type ThemeOriginCode,
} from '@numen/core-client'
import type { PaletteGroup, PaletteItem } from '@numen/shell'
import { WORDS } from '@/shared/words'

/** Which half of a `light-dark()` pair every token is read as. */
export type Mode = 'system' | 'light' | 'dark'

/** The modes, in the order they are offered. */
export const MODES: readonly Mode[] = ['system', 'light', 'dark']

/** What `color-scheme` is written as for each mode. */
const SCHEMES: Record<Mode, string> = { system: 'light dark', light: 'light', dark: 'dark' }

/** The mode as the schema carries it. */
const MODE_CODES: Record<Mode, ThemeModeCode> = {
  system: THEME_MODE.SYSTEM,
  light: THEME_MODE.LIGHT,
  dark: THEME_MODE.DARK,
}

/** The attribute the page's style elements carry, and the two this window writes. */
const MARKER = 'data-appearance'
const IS_MODE = 'mode'
const IS_THEME = 'theme'

/** The prefix of the identity of a theme item and of a mode item. */
const THEME_PREFIX = 'theme:'
const MODE_PREFIX = 'mode:'

/** One theme as the application lists it. */
export interface ThemeEntry {
  readonly name: string
  readonly title: string
  readonly origin: ThemeOriginCode
  readonly pluginId: string
  /** The theme declares light and dark itself, so the mode has nothing left to choose. */
  readonly isPinned: boolean
}

/** The themes and the choice the application reports. */
export interface ThemeList {
  readonly themes: readonly ThemeEntry[]
  readonly applied: string
  readonly mode: ThemeModeCode
}

/** The theme service as the window uses it. */
export interface ThemeClient {
  listThemes(): Promise<ThemeList>
  readTheme(name: string): Promise<string>
  /** Writes the choice, and answers why it was not written; empty where it was. */
  writeAppearance(name: string, mode: ThemeModeCode): Promise<string>
  /** The names of themes that changed, until the signal aborts. */
  watchThemes(signal: AbortSignal): AsyncIterable<readonly string[]>
}

export interface AppearanceDeps {
  readonly client: ThemeClient
  /** The name a plugin is listed under, given its id. */
  readonly getPluginName: (pluginId: string) => string
  /** Reports a failure to the window's notices. */
  readonly onError: (text: string) => void
  readonly document?: Document
}

/** The style element of the page marked as one of the two, created when the page carries none. */
function findStyle(is: string, page: Document): HTMLStyleElement {
  const found = page.head.querySelector<HTMLStyleElement>(`style[${MARKER}="${is}"]`)
  if (found) return found
  const created = page.createElement('style')
  created.setAttribute(MARKER, is)
  page.head.appendChild(created)
  return created
}

/** The mode in the window's own words; a mode with no word is the system's. */
function getModeWord(code: ThemeModeCode): Mode {
  return MODES.find((word) => MODE_CODES[word] === code) ?? 'system'
}

interface AppearanceState {
  readonly themes: Readonly<ShallowRef<readonly ThemeEntry[]>>
  readonly applied: Readonly<Ref<string>>
  readonly mode: Readonly<Ref<Mode>>
  readonly getPluginName: (pluginId: string) => string
}

/** The palette groups the themes and the modes make, filtered by the text typed. */
function createPaletteGroups({ themes, applied, mode, getPluginName }: AppearanceState) {
  const isPinned = () => themes.value.find((one) => one.name === applied.value)?.isPinned ?? false

  const getThemeItem = (one: ThemeEntry): PaletteItem => ({
    id: THEME_PREFIX + one.name,
    title: one.title,
    ...(one.name === applied.value ? { detail: WORDS.current } : {}),
    actions: [{ id: 'apply', text: WORDS.apply }],
  })

  /** The theme groups: the ones the application ships, then each plugin's, under its name. */
  const getThemeGroups = (): readonly PaletteGroup[] => {
    const shipped = themes.value.filter((one) => one.origin === THEME_ORIGIN.PRESET)
    const own = themes.value.filter((one) => one.origin === THEME_ORIGIN.MINE)
    const pluginIds = [
      ...new Set(themes.value.filter((one) => one.pluginId !== '').map((one) => one.pluginId)),
    ]
    return [
      { id: 'shipping', title: WORDS.shipping, items: shipped.map(getThemeItem) },
      ...pluginIds.map((pluginId) => ({
        id: `plugin:${pluginId}`,
        title: getPluginName(pluginId),
        items: themes.value.filter((one) => one.pluginId === pluginId).map(getThemeItem),
      })),
      { id: 'owned', title: WORDS.owned, items: own.map(getThemeItem) },
    ]
  }

  /** The three modes. None can be chosen while the theme worn pins light or dark. */
  const getModeGroups = (): readonly PaletteGroup[] => [
    {
      id: 'half',
      title: WORDS.half,
      items: MODES.map((word) => ({
        id: MODE_PREFIX + word,
        title: WORDS.modes[word],
        ...(isPinned() ? { detail: WORDS.pinned, isDisabled: true } : {}),
        ...(!isPinned() && word === mode.value ? { detail: WORDS.current } : {}),
        actions: [{ id: 'apply', text: WORDS.apply }],
      })),
    },
  ]

  /** The groups whose items contain the text, ignoring case. */
  return (typed: string): readonly PaletteGroup[] => {
    const needle = typed.trim().toLowerCase()
    return [...getThemeGroups(), ...getModeGroups()].map((group) => ({
      ...group,
      items: group.items.filter((one) => one.title.toLowerCase().includes(needle)),
    }))
  }
}

/** The state of the appearance, the palette groups it offers, and the choices made from them. */
export function createAppearance(deps: AppearanceDeps) {
  const { client, getPluginName, onError, document: page = document } = deps
  const themes = shallowRef<readonly ThemeEntry[]>([])
  const applied = ref('')
  const mode = ref<Mode>('system')

  /** The page dressed in a theme and a mode. */
  const dress = async (name: string, half: Mode) => {
    const css = await client.readTheme(name)
    findStyle(IS_MODE, page).textContent = `:root { color-scheme: ${SCHEMES[half]}; }`
    findStyle(IS_THEME, page).textContent = css
  }

  /** The themes and the choice read again, and the page dressed where the choice changed. */
  const refresh = async () => {
    const before = { applied: applied.value, mode: mode.value }
    try {
      const answer = await client.listThemes()
      themes.value = answer.themes
      applied.value = answer.applied
      mode.value = getModeWord(answer.mode)
      const isChanged = before.applied !== '' && before.applied !== applied.value
      if (isChanged || before.mode !== mode.value) await dress(applied.value, mode.value)
    } catch (error) {
      onError(`${WORDS.themesUnlisted} ${formatErrorMessage(error)}`)
    }
  }

  /** Writes a choice, and dresses the page in it where it was written. */
  const choose = async (name: string, half: Mode) => {
    try {
      const reason = await client.writeAppearance(name, MODE_CODES[half])
      if (reason !== '') {
        onError(reason)
        return
      }
      applied.value = name
      mode.value = half
      await dress(name, half)
    } catch (error) {
      onError(formatErrorMessage(error))
    }
  }

  const getGroups = createPaletteGroups({ themes, applied, mode, getPluginName })

  /** Applies the item the palette reports chosen; an item of neither kind changes nothing. */
  const chooseItem = async (id: string) => {
    if (id.startsWith(THEME_PREFIX)) {
      await choose(id.slice(THEME_PREFIX.length), mode.value)
      return
    }
    const half = MODES.find((word) => MODE_PREFIX + word === id)
    if (half) await choose(applied.value, half)
  }

  return { themes, applied, mode, refresh, getGroups, chooseItem }
}
