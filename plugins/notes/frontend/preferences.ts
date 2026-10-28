/**
 * Editor preferences read from the notes plugin settings and kept reactive while active.
 */
import { reactive } from 'vue'
import type { PluginContext } from '@numen/plugin-sdk'

export interface MarkdownEditorPreferences {
  softWrap?: boolean
  lineNumbers?: boolean
  tabSize?: number
}

export type SettingLookup =
  | ((key: string) => unknown)
  | ((path: readonly string[]) => unknown)

function readSetting(getSetting: SettingLookup | undefined, key: string): unknown {
  if (!getSetting) return undefined
  const byKey = (getSetting as (arg: string) => unknown)(key)
  if (byKey !== undefined) return byKey
  return (getSetting as (arg: readonly string[]) => unknown)([key])
}

export function readMarkdownEditorPreferences(
  getSetting?: SettingLookup,
): Required<MarkdownEditorPreferences> {
  const softWrap = readSetting(getSetting, 'soft_wrap')
  const lineNumbers = readSetting(getSetting, 'line_numbers')
  const tabSize = readSetting(getSetting, 'tab_size')
  return {
    softWrap: typeof softWrap === 'boolean' ? softWrap : true,
    lineNumbers: typeof lineNumbers === 'boolean' ? lineNumbers : true,
    tabSize: typeof tabSize === 'number' && Number.isFinite(tabSize) ? Math.max(1, Math.trunc(tabSize)) : 2,
  }
}

export const editorPreferences = reactive<Required<MarkdownEditorPreferences>>({
  softWrap: true,
  lineNumbers: false,
  tabSize: 2,
})

export function bindMarkdownEditorPreferences(
  context: Pick<Partial<PluginContext>, 'getSetting' | 'onSettingChange'>,
): Required<MarkdownEditorPreferences> {
  const applyAll = () => {
    const next = readMarkdownEditorPreferences(context.getSetting)
    editorPreferences.softWrap = next.softWrap
    editorPreferences.lineNumbers = next.lineNumbers
    editorPreferences.tabSize = next.tabSize
  }
  applyAll()

  context.onSettingChange?.('soft_wrap', (value) => {
    editorPreferences.softWrap =
      typeof value === 'boolean'
        ? value
        : readMarkdownEditorPreferences(context.getSetting).softWrap
  })
  context.onSettingChange?.('line_numbers', (value) => {
    editorPreferences.lineNumbers =
      typeof value === 'boolean'
        ? value
        : readMarkdownEditorPreferences(context.getSetting).lineNumbers
  })
  context.onSettingChange?.('tab_size', (value) => {
    editorPreferences.tabSize =
      typeof value === 'number' && Number.isFinite(value)
        ? Math.max(1, Math.trunc(value))
        : readMarkdownEditorPreferences(context.getSetting).tabSize
  })

  return editorPreferences
}
