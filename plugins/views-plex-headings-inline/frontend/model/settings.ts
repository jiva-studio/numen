/** How many headings a node shows, as the plugin's own setting holds it. */
import { shallowRef, type Ref } from 'vue'
import type { JsonValue, PluginContext } from '@numen/plugin-sdk'
const HEADING_LIMIT = 6
const HEADING_LIMIT_MIN = 1
const HEADING_LIMIT_MAX = 12

/** The setting, kept in sync as the person changes it. */
export interface HeadingsSettings {
  readonly headingLimit: Readonly<Ref<number>>
}

const parseHeadingLimit = (value: unknown): number =>
  typeof value === 'number' && Number.isInteger(value)
    ? Math.min(Math.max(value, HEADING_LIMIT_MIN), HEADING_LIMIT_MAX)
    : HEADING_LIMIT

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
export function readSettings(context: Pick<PluginContext, 'settings'>): HeadingsSettings {
  return {
    headingLimit: follow(context, 'headingLimit', parseHeadingLimit),
  }
}
