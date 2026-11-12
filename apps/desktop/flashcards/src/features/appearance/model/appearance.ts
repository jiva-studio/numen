/**
 * The appearance commands of the window: the themes the application lists and the three modes of
 * a color pair, offered as items of the shell's command palette. Choosing one writes it to the
 * settings through the theme service and applies it to the page.
 */
import { ref, shallowRef, type Ref, type ShallowRef } from 'vue'
import type { PaletteGroup, PaletteItem } from '@numen/workbench'
import { formatError } from '@/shared/api/errors'
import type { ThemeClient } from '../api/theme'
import { MODES, type Mode, type ThemeEntry } from '../lib/appearance'
import { WORDS } from '../words'

/** What `color-scheme` is written as for each mode. */
const SCHEMES: Record<Mode, string> = { system: 'light dark', light: 'light', dark: 'dark' }

/** The attribute the page's style elements carry, and the two this window writes. */
const MARKER = 'data-appearance'
const IS_MODE = 'mode'
const IS_THEME = 'theme'

/** The prefix of the identity of a theme item and of a mode item. */
const THEME_PREFIX = 'theme:'
const MODE_PREFIX = 'mode:'

interface AppearanceDeps {
  readonly client: ThemeClient
  readonly getPluginName: (pluginId: string) => string
  readonly onError: (text: string) => void
  readonly document?: Document
}

function findStyle(is: string, page: Document): HTMLStyleElement {
  const found = page.head.querySelector<HTMLStyleElement>(`style[${MARKER}="${is}"]`)
  if (found) return found
  const created = page.createElement('style')
  created.setAttribute(MARKER, is)
  page.head.appendChild(created)
  return created
}

interface AppearanceState {
  readonly themes: Readonly<ShallowRef<readonly ThemeEntry[]>>
  readonly applied: Readonly<Ref<string>>
  readonly mode: Readonly<Ref<Mode>>
  readonly getPluginName: (pluginId: string) => string
}

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
    const shipped = themes.value.filter((one) => one.origin === 'preset')
    const own = themes.value.filter((one) => one.origin === 'mine')
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

  /** The three modes. None can be chosen while the applied theme pins light or dark. */
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
  const { client, getPluginName, onError, document: page = globalThis.document } = deps
  const themes = shallowRef<readonly ThemeEntry[]>([])
  const applied = ref('')
  const mode = ref<Mode>('system')

  /** The page with a theme and a mode applied. */
  const applyTheme = async (name: string, half: Mode) => {
    const css = await client.readTheme(name)
    findStyle(IS_MODE, page).textContent = `:root { color-scheme: ${SCHEMES[half]}; }`
    findStyle(IS_THEME, page).textContent = css
  }

  /** The themes and the choice read again, and the theme and mode applied to the page where the choice changed. */
  const refresh = async () => {
    const before = { applied: applied.value, mode: mode.value }
    try {
      const answer = await client.listThemes()
      themes.value = answer.themes
      applied.value = answer.applied
      mode.value = answer.mode
      const isChanged = before.applied !== '' && before.applied !== applied.value
      if (isChanged || before.mode !== mode.value) await applyTheme(applied.value, mode.value)
    } catch (error) {
      onError(`${WORDS.themesUnlisted} ${formatError(error)}`)
    }
  }

  /** Writes a choice, and applies it to the page where it was written. */
  const choose = async (name: string, half: Mode) => {
    try {
      const reason = await client.writeAppearance(name, half)
      if (reason !== '') {
        onError(reason)
        return
      }
      applied.value = name
      mode.value = half
      await applyTheme(name, half)
    } catch (error) {
      onError(formatError(error))
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
