/** How many names a note shows, as the plugin's own settings hold it. */
import { shallowRef, type Ref } from 'vue'
import type { JsonValue, PluginContext, Unvalidated } from '@numen/plugin-sdk'

const NAMES_PER_NOTE = 6
const NAMES_PER_NOTE_MIN = 1
const NAMES_PER_NOTE_MAX = 12

/** The settings, kept in sync as the person changes them. */
export interface NamesSettings {
  readonly namesPerNote: Readonly<Ref<number>>
}

const parseNamesPerNote = (value: Unvalidated): number =>
  typeof value === 'number' && Number.isFinite(value)
    ? Math.min(Math.max(Math.round(value), NAMES_PER_NOTE_MIN), NAMES_PER_NOTE_MAX)
    : NAMES_PER_NOTE

function follow<T>(
  context: Pick<PluginContext, 'settings'>,
  key: string,
  parse: (value: JsonValue | undefined) => T,
): Ref<T> {
  const value = shallowRef(parse(context.settings.get(key)))
  context.settings.onChange(key, (next) => {
    value.value = parse(next)
  })
  return value
}

/** The settings read from the context, and followed while the plugin is active. */
export function readSettings(context: Pick<PluginContext, 'settings'>): NamesSettings {
  return {
    namesPerNote: follow(context, 'namesPerNote', parseNamesPerNote),
  }
}
