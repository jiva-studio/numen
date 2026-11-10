/**
 * Editor preferences read from the notes plugin settings and kept reactive while active.
 */
import { reactive } from 'vue'
import type { PluginContext } from '@numen/plugin-sdk'

export interface MarkdownEditorPreferences {
  softWrap?: boolean
  lineNumbers?: boolean
}

export type SettingLookup = ((key: string) => unknown) | ((path: readonly string[]) => unknown)

function readSetting(getSetting: SettingLookup | undefined, key: string): unknown {
  if (!getSetting) return undefined
  const byKey = (getSetting as (arg: string) => unknown)(key)
  if (byKey !== undefined) return byKey
  return (getSetting as (arg: readonly string[]) => unknown)([key])
}

export function readMarkdownEditorPreferences(
  getSetting?: SettingLookup,
): Required<MarkdownEditorPreferences> {
  const softWrap = readSetting(getSetting, 'softWrap')
  const lineNumbers = readSetting(getSetting, 'lineNumbers')
  return {
    softWrap: typeof softWrap === 'boolean' ? softWrap : true,
    lineNumbers: typeof lineNumbers === 'boolean' ? lineNumbers : false,
  }
}

export const editorPreferences = reactive<Required<MarkdownEditorPreferences>>({
  softWrap: true,
  lineNumbers: false,
})

export function bindMarkdownEditorPreferences(
  context: Pick<Partial<PluginContext>, 'settings'>,
): Required<MarkdownEditorPreferences> {
  const applyAll = () => {
    const next = readMarkdownEditorPreferences(context.settings?.get)
    editorPreferences.softWrap = next.softWrap
    editorPreferences.lineNumbers = next.lineNumbers
  }
  applyAll()

  context.settings?.onChange.('softWrap', (value) => {
    editorPreferences.softWrap =
      typeof value === 'boolean'
        ? value
        : readMarkdownEditorPreferences(context.settings?.get).softWrap
  })
  context.settings?.onChange.('lineNumbers', (value) => {
    editorPreferences.lineNumbers =
      typeof value === 'boolean'
        ? value
        : readMarkdownEditorPreferences(context.settings?.get).lineNumbers
  })
  return editorPreferences
}
