/**
 * The texts of a plugin's views, read and edited over TextService.
 */
import {
  TEXT_ERROR_CODE,
  type TextClient,
  type TextSegmentMessage,
  type TextVersionMessage,
} from '@numen/core-client'
import type {
  JsonObject,
  JsonValue,
  LiveText,
  Selector,
  TextSegment,
  TextVersion,
  VaultTexts,
} from '@numen/plugin-sdk'

const readSelector = (raw: string): Selector | undefined => {
  try {
    const parsed: JsonValue = JSON.parse(raw)
    return typeof parsed === 'object' && parsed !== null && !Array.isArray(parsed)
      ? (parsed as JsonObject)
      : undefined
  } catch {
    return undefined
  }
}

const readSegment = (one: TextSegmentMessage): TextSegment => {
  const selector = readSelector(one.selector)
  return {
    text: one.text,
    ...(one.heading === '' ? {} : { heading: one.heading }),
    ...(selector === undefined ? {} : { selector }),
  }
}

const readVersion = (one: TextVersionMessage): TextVersion => ({
  id: one.id,
  path: one.path,
  pluginId: one.pluginId,
  key: one.key,
  sourceVersionId: one.sourceVersionId,
  ...(one.sourceProperty === undefined || one.sourceProperty.key === ''
    ? {}
    : { sourceProperty: { key: one.sourceProperty.key, value: one.sourceProperty.value } }),
  kind: one.kind,
  file: one.file,
  createTime: new Date(Number(one.createTimeUnixMs)),
})

const writeSegment = (one: TextSegment) => ({
  text: one.text,
  heading: one.heading ?? '',
  selector: one.selector === undefined ? '' : JSON.stringify(one.selector),
})

/** The texts a plugin's views read, and edit in the plugin's folder. */
export const createPluginTexts = (client: TextClient, pluginId: string): VaultTexts => ({
  readLiveText: async (path): Promise<LiveText | null> => {
    const answer = await client.readLiveText({ path })
    if (answer.version === undefined) return null
    return { version: readVersion(answer.version), segments: answer.segments.map(readSegment) }
  },
  createEdit: async (path, sourceVersionId, segments) => {
    const answer = await client.createTextEdit({
      pluginId,
      path,
      sourceVersionId,
      segments: segments.map(writeSegment),
    })
    if (answer.version !== undefined && answer.error === TEXT_ERROR_CODE.UNSPECIFIED) {
      return { ok: true, value: readVersion(answer.version) }
    }
    return { ok: false, error: answer.error === TEXT_ERROR_CODE.MISSING ? 'missing' : 'invalid' }
  },
})
