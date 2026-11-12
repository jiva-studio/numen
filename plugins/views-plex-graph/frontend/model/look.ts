/**
 * How the plex is drawn, as the plugin's own settings hold it: the pills, the size of the
 * focus, the lines, the movement and the room around a note.
 */
import { shallowRef, type Ref } from 'vue'
import type { PluginContext, Unvalidated } from '@numen/plugin-sdk'
import type { PlexLook } from '../types'

/** The key the manifest declares Pill background under. */
const SHOW_PILLS = 'showPills'

/** A number setting: the default the manifest declares, and the range it takes. */
interface NumberRange {
  readonly fallback: number
  readonly min: number
  readonly max: number
}

/** The number settings under the keys the manifest declares them. */
const LOOK_NUMBERS = {
  pillPaddingSides: { fallback: 10, min: 0, max: 24 },
  pillPaddingEnds: { fallback: 9.25, min: 0, max: 16 },
  focusSize: { fallback: 1.22, min: 1, max: 2.5 },
  animationLength: { fallback: 420, min: 0, max: 2000 },
  lineThickness: { fallback: 2, min: 0.5, max: 4 },
  outlineThickness: { fallback: 1, min: 0.5, max: 4 },
  lineCurvature: { fallback: 0.55, min: 0, max: 1 },
  noteRoom: { fallback: 0, min: 0, max: 40 },
} as const satisfies Record<string, NumberRange>

type LookNumber = keyof typeof LOOK_NUMBERS

const parseShowPills = (value: Unvalidated): boolean => (typeof value === 'boolean' ? value : true)

/** A finite number clamped into its range; anything else is the default. */
const parseNumber = (value: Unvalidated, range: NumberRange): number =>
  typeof value === 'number' && Number.isFinite(value)
    ? Math.min(Math.max(value, range.min), range.max)
    : range.fallback

function followSetting<T>(
  context: Pick<PluginContext, 'settings'>,
  key: string,
  parse: (value: Unvalidated) => T,
): Ref<T> {
  const value = shallowRef(parse(context.settings.get(key)))
  context.settings.onChange(key, (next) => {
    value.value = parse(next)
  })
  return value
}

/** The look read from the context, and followed while the plugin is active. */
export function readLook(context: Pick<PluginContext, 'settings'>): PlexLook {
  const numbers = Object.fromEntries(
    Object.entries(LOOK_NUMBERS).map(([key, range]) => [
      key,
      followSetting(context, key, (value) => parseNumber(value, range)),
    ]),
  ) as Record<LookNumber, Ref<number>>
  return { ...numbers, showPills: followSetting(context, SHOW_PILLS, parseShowPills) }
}

/** The look at its defaults, for a plex no settings are read for. */
export const createDefaultLook = (): PlexLook =>
  readLook({ settings: { get: () => undefined, onChange: () => ({ dispose: () => undefined }) } })
