/**
 * Whether a node nests the headings of its note, and how many of them it shows,
 * as the plugin's own settings hold them.
 */
import { ref, type Ref } from 'vue'
import type { PluginContext, Unvalidated } from '@numen/plugin-sdk'

/** The keys the manifest declares the two settings under. */
export const NEST_HEADINGS = 'nestHeadings'
export const HEADING_LIMIT = 'headingLimit'

/** The heading limit the manifest declares as its default, and the range it takes. */
export const DEFAULT_HEADING_LIMIT = 6
export const HEADING_LIMIT_MIN = 1
export const HEADING_LIMIT_MAX = 12

/** The two settings, kept in sync as the user changes them. */
export interface Nesting {
  readonly isNesting: Readonly<Ref<boolean>>
  readonly headings: Readonly<Ref<number>>
}

const parseIsNesting = (value: Unvalidated): boolean => typeof value === 'boolean' ? value : true

const parseHeadingLimit = (value: Unvalidated): number =>
  typeof value === 'number' && Number.isInteger(value)
    ? Math.min(Math.max(value, HEADING_LIMIT_MIN), HEADING_LIMIT_MAX)
    : DEFAULT_HEADING_LIMIT

/** The two settings read from the context, and followed while the plugin is active. */
export function readNesting(context: Pick<PluginContext, 'settings'>): Nesting {
  const isNesting = ref(parseIsNesting(context.settings.get(NEST_HEADINGS)))
  const headings = ref(parseHeadingLimit(context.settings.get(HEADING_LIMIT)))
  context.settings.onChange(NEST_HEADINGS, (value) => {
    isNesting.value = parseIsNesting(value)
  })
  context.settings.onChange(HEADING_LIMIT, (value) => {
    headings.value = parseHeadingLimit(value)
  })
  return { isNesting, headings }
}
