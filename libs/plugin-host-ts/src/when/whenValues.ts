/**
 * The values a condition is evaluated over, built in this one place from what a window knows: its
 * window variables, the target and the note read at it, the content types the core lists, and the
 * settings as `config`.
 */
import { getTargetValues, type TargetNote, type TargetPath } from './targetValues'
import type { WhenSettings, WhenValues } from './when'

/** What the values of one evaluation are made of. */
export interface WhenValueParts {
  readonly window?: WhenValues | undefined
  readonly target: TargetPath | undefined
  readonly note?: TargetNote | undefined
  readonly contentTypes: Readonly<Record<string, string>>
  readonly config?: WhenSettings | undefined
}

/** The values of the window, the target and the settings; a part left out gives its variables none. */
export function getWhenValues(parts: WhenValueParts): WhenValues {
  return {
    ...parts.window,
    ...getTargetValues(parts.target, parts.contentTypes, parts.note),
    ...(parts.config === undefined ? {} : { config: parts.config }),
  }
}
