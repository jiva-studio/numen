/**
 * Editor preferences read from the notes plugin settings and kept reactive while active.
 */
import { reactive } from 'vue'
import type { JsonValue, PluginContext } from '@numen/plugin-sdk'
import { getFontFamilyValue } from './fontFamily'

export interface MarkdownEditorPreferences {
  softWrap?: boolean
  lineNumbers?: boolean
  tabSize?: number
  fontFamily?: string
  lineHeight?: number
  highlightActiveLine?: boolean
}

export type SettingLookup = (key: string) => JsonValue | undefined

type Preferences = Required<MarkdownEditorPreferences>

export const DEFAULT_PREFERENCES: Readonly<Preferences> = {
  softWrap: true,
  lineNumbers: false,
  tabSize: 2,
  fontFamily: '',
  lineHeight: 1.6,
  highlightActiveLine: false,
}

const isBetween = (value: JsonValue | undefined, min: number, max: number): value is number =>
  typeof value === 'number' && Number.isFinite(value) && value >= min && value <= max

const isValid: { [K in keyof Preferences]: (value: JsonValue | undefined) => boolean } = {
  softWrap: (value) => typeof value === 'boolean',
  lineNumbers: (value) => typeof value === 'boolean',
  tabSize: (value) => isBetween(value, 1, 8) && Number.isInteger(value),
  fontFamily: (value) => typeof value === 'string' && getFontFamilyValue(value) !== null,
  lineHeight: (value) => isBetween(value, 1.2, 2.4),
  highlightActiveLine: (value) => typeof value === 'boolean',
}

const KEYS = Object.keys(DEFAULT_PREFERENCES) as (keyof Preferences)[]

export function readMarkdownEditorPreferences(getSetting?: SettingLookup): Preferences {
  const next: Preferences = { ...DEFAULT_PREFERENCES }
  for (const key of KEYS) {
    const value = getSetting?.(key)
    if (isValid[key](value)) Object.assign(next, { [key]: value })
  }
  return next
}

export const editorPreferences = reactive<Preferences>({ ...DEFAULT_PREFERENCES })

export function bindMarkdownEditorPreferences(
  context: Pick<Partial<PluginContext>, 'settings'>,
): Preferences {
  Object.assign(editorPreferences, readMarkdownEditorPreferences(context.settings?.get))

  for (const key of KEYS) {
    context.settings?.onChange(key, (value) => {
      const next = isValid[key](value)
        ? value
        : readMarkdownEditorPreferences(context.settings?.get)[key]
      Object.assign(editorPreferences, { [key]: next })
    })
  }
  return editorPreferences
}
