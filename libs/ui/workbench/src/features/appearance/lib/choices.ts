/**
 * The appearance as choices of a palette: the themes the application lists, grouped by where they
 * come from, and the three modes of a color pair. The groups are shaped as the palette draws them.
 */
import { MODES, type Mode, type Theme } from './appearance'

export interface AppearanceChoiceWords {
  readonly apply: string
  readonly shipping: string
  readonly owned: string
  readonly half: string
  readonly current: string
  readonly pinned: string
  readonly modes: Readonly<Record<Mode, string>>
}

/** One choice, as a palette item. */
export interface AppearanceChoice {
  readonly id: string
  readonly title: string
  readonly detail?: string
  readonly isDisabled?: boolean
  readonly actions: readonly { readonly id: string; readonly text: string }[]
}

/** One group of choices, as a palette group. */
export interface AppearanceChoiceGroup {
  readonly id: string
  readonly title: string
  readonly items: readonly AppearanceChoice[]
}

/** What the choices are read from. */
export interface AppearanceChoiceState {
  readonly themes: readonly Theme[]
  readonly applied: string
  readonly mode: Mode
  readonly getPluginName: (pluginId: string) => string
}

/** What a choice picked: a theme by name, or a mode. */
export type AppearancePick =
  | { readonly kind: 'theme'; readonly theme: string }
  | { readonly kind: 'mode'; readonly mode: Mode }

const THEME_PREFIX = 'theme:'
const MODE_PREFIX = 'mode:'

function getThemeGroups(state: AppearanceChoiceState, words: AppearanceChoiceWords) {
  const { themes, applied, getPluginName } = state
  const getThemeItem = (one: Theme): AppearanceChoice => ({
    id: THEME_PREFIX + one.name,
    title: one.title,
    ...(one.name === applied ? { detail: words.current } : {}),
    actions: [{ id: 'apply', text: words.apply }],
  })
  const pluginIds = [
    ...new Set(themes.filter((one) => one.pluginId !== '').map((one) => one.pluginId)),
  ]
  return [
    {
      id: 'shipping',
      title: words.shipping,
      items: themes.filter((one) => one.origin === 'preset').map(getThemeItem),
    },
    ...pluginIds.map((pluginId) => ({
      id: `plugin:${pluginId}`,
      title: getPluginName(pluginId),
      items: themes.filter((one) => one.pluginId === pluginId).map(getThemeItem),
    })),
    {
      id: 'owned',
      title: words.owned,
      items: themes.filter((one) => one.origin === 'mine').map(getThemeItem),
    },
  ]
}

function getModeGroup(
  state: AppearanceChoiceState,
  words: AppearanceChoiceWords,
): AppearanceChoiceGroup {
  const isPinned = state.themes.find((one) => one.name === state.applied)?.isPinned ?? false
  return {
    id: 'half',
    title: words.half,
    items: MODES.map((word) => ({
      id: MODE_PREFIX + word,
      title: words.modes[word],
      ...(isPinned ? { detail: words.pinned, isDisabled: true } : {}),
      ...(!isPinned && word === state.mode ? { detail: words.current } : {}),
      actions: [{ id: 'apply', text: words.apply }],
    })),
  }
}

/**
 * The theme groups, the ones the application ships, then each plugin's under its name, then the
 * person's own, and the mode group, each with the choices whose title contains the text, ignoring case.
 * No mode can be chosen while the applied theme pins light or dark.
 */
export function getAppearanceGroups(
  state: AppearanceChoiceState,
  words: AppearanceChoiceWords,
  typed: string,
): readonly AppearanceChoiceGroup[] {
  const needle = typed.trim().toLowerCase()
  return [...getThemeGroups(state, words), getModeGroup(state, words)].map((group) => ({
    ...group,
    items: group.items.filter((one) => one.title.toLowerCase().includes(needle)),
  }))
}

/** What the choice of that identity picks; undefined for an identity of neither kind. */
export function readAppearancePick(id: string): AppearancePick | undefined {
  if (id.startsWith(THEME_PREFIX)) return { kind: 'theme', theme: id.slice(THEME_PREFIX.length) }
  const mode = MODES.find((word) => MODE_PREFIX + word === id)
  return mode ? { kind: 'mode', mode } : undefined
}
