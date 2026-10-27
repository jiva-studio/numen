/**
 * Whether a node nests the headings of its note, and how many of them it shows,
 * as the plugin's own settings hold them.
 */
import { ref, type Ref } from 'vue'
import type { PluginContext } from '@numen/plugin-sdk'

/** The keys the manifest declares the two settings under. */
export const NEST_HEADINGS = 'nest_headings'
export const HEADING_LIMIT = 'heading_limit'

/** The heading limit the manifest declares as its default, and the range it takes. */
export const DEFAULT_HEADING_LIMIT = 6
export const HEADING_LIMIT_MIN = 1
export const HEADING_LIMIT_MAX = 12

/** The two settings, kept current as the person changes them. */
export interface Nesting {
  readonly isNesting: Readonly<Ref<boolean>>
  readonly headings: Readonly<Ref<number>>
}

/** An unset value nests the headings. */
export const parseIsNesting = (value: unknown): boolean => value !== false

/** A value that is no whole number is the default; a number outside the range is clamped to it. */
export const parseHeadingLimit = (value: unknown): number =>
  typeof value === 'number' && Number.isInteger(value)
    ? Math.min(Math.max(value, HEADING_LIMIT_MIN), HEADING_LIMIT_MAX)
    : DEFAULT_HEADING_LIMIT

/** The two settings read from the context, and followed while the plugin is active. */
export function readNesting(
  context: Pick<PluginContext, 'getSetting' | 'onSettingChange'>,
): Nesting {
  const isNesting = ref(parseIsNesting(context.getSetting(NEST_HEADINGS)))
  const headings = ref(parseHeadingLimit(context.getSetting(HEADING_LIMIT)))
  context.onSettingChange(NEST_HEADINGS, (value) => {
    isNesting.value = parseIsNesting(value)
  })
  context.onSettingChange(HEADING_LIMIT, (value) => {
    headings.value = parseHeadingLimit(value)
  })
  return { isNesting, headings }
}
