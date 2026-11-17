/** Whether a node nests the headings of its note and how many it shows, as the plugin's own settings hold them. */
import { shallowRef, type Ref } from 'vue'
import type { JsonValue, PluginContext, Unvalidated } from '@numen/plugin-sdk'

const NEST_HEADINGS = true
const HEADING_LIMIT = 6
const HEADING_LIMIT_MIN = 1
const HEADING_LIMIT_MAX = 12

/** The two settings, kept in sync as the person changes them. */
export interface HeadingsSettings {
  readonly isNesting: Readonly<Ref<boolean>>
  readonly headingLimit: Readonly<Ref<number>>
}

const parseIsNesting = (value: Unvalidated): boolean =>
  typeof value === 'boolean' ? value : NEST_HEADINGS

const parseHeadingLimit = (value: Unvalidated): number =>
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
    isNesting: follow(context, 'nestHeadings', parseIsNesting),
    headingLimit: follow(context, 'headingLimit', parseHeadingLimit),
  }
}
