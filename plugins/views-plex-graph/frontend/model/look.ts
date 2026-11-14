/**
 * How the plex is drawn, as the plugin's own settings hold it: the pills, the size of the
 * focus, the lines, the movement and the room around a note.
 */
import type { Ref } from 'vue'
import type { PluginContext, Unvalidated } from '@numen/plugin-sdk'
import { followSetting, NO_SETTINGS } from './followSetting'
import { DEFAULT_FOCUSED_NAMES_OPACITY, DEFAULT_NAMES_OPACITY } from '../plex'
import type { PlexLook, ReferenceMode } from '../types'

/** The key the manifest declares Pill background under. */
const SHOW_PILLS = 'showPills'

/** The key the manifest declares Highlight links of a hovered note under. */
const HIGHLIGHT_LINKS = 'highlightLinks'

/** The key the manifest declares References under. */
const REFERENCES = 'references'

/** A number setting: the default the manifest declares, and the range it takes. */
interface NumberRange {
  readonly fallback: number
  readonly min: number
  readonly max: number
}

/** The number settings under the keys the manifest declares them. */
const LOOK_NUMBERS = [
  ['pillPaddingSides', { fallback: 10, min: 0, max: 24 }],
  ['pillPaddingEnds', { fallback: 9.25, min: 0, max: 16 }],
  ['focusSize', { fallback: 1.22, min: 1, max: 2.5 }],
  ['animationLength', { fallback: 420, min: 0, max: 2000 }],
  ['lineThickness', { fallback: 2, min: 0.5, max: 4 }],
  ['outlineThickness', { fallback: 1, min: 0.5, max: 4 }],
  ['lineCurvature', { fallback: 0.55, min: 0, max: 1 }],
  ['noteRoom', { fallback: 0, min: 0, max: 40 }],
  ['namesOpacity', { fallback: DEFAULT_NAMES_OPACITY, min: 0, max: 1 }],
  ['focusedNamesOpacity', { fallback: DEFAULT_FOCUSED_NAMES_OPACITY, min: 0.05, max: 1 }],
] as const satisfies readonly (readonly [string, NumberRange])[]

type LookNumber = (typeof LOOK_NUMBERS)[number][0]

const REFERENCE_MODES: readonly ReferenceMode[] = ['off', 'outgoing', 'incoming', 'both']

const parseReferences = (value: Unvalidated): ReferenceMode =>
  typeof value === 'string' ? (REFERENCE_MODES.find((mode) => mode === value) ?? 'both') : 'both'

const parseShowPills = (value: Unvalidated): boolean => (typeof value === 'boolean' ? value : true)

const parseHighlightLinks = (value: Unvalidated): boolean =>
  typeof value === 'boolean' ? value : true

const parseNumber = (value: Unvalidated, range: NumberRange): number =>
  typeof value === 'number' && Number.isFinite(value)
    ? Math.min(Math.max(value, range.min), range.max)
    : range.fallback

/** The look read from the context, and followed while the plugin is active. */
export function readLook(context: Pick<PluginContext, 'settings'>): PlexLook {
  const numbers = Object.fromEntries(
    LOOK_NUMBERS.map(([key, range]) => [
      key,
      followSetting(context, key, (value) => parseNumber(value, range)),
    ]),
  ) as Record<LookNumber, Ref<number>>
  return {
    ...numbers,
    showPills: followSetting(context, SHOW_PILLS, parseShowPills),
    highlightLinks: followSetting(context, HIGHLIGHT_LINKS, parseHighlightLinks),
    references: followSetting(context, REFERENCES, parseReferences),
  }
}

/** The look at its defaults, for a plex no settings are read for. */
export const getDefaultLook = (): PlexLook => readLook(NO_SETTINGS)
