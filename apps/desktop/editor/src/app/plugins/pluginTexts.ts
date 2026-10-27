/**
 * The text versions of the shown vault, as the views of one plugin read them and save an edit.
 */
import type { LiveText, Selector, TextSegment, TextVersion, VaultTexts } from '@numen/plugin-sdk'
import type { Client } from '@connectrpc/connect'
import {
  TextErrorCode,
  type TextService,
  type TextSegment as TextSegmentMessage,
  type TextVersion as TextVersionMessage,
} from '@numen/protocol'

/** The calls of TextService the texts are read and edited through. */
export type TextClient = Pick<Client<typeof TextService>, 'readLiveText' | 'createTextEdit'>

const readSelector = (raw: string): Selector | undefined => {
  if (raw === '') return undefined
  try {
    const parsed: unknown = JSON.parse(raw)
    return typeof parsed === 'object' && parsed !== null ? (parsed as Selector) : undefined
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
    if (answer.version !== undefined && answer.error === TextErrorCode.UNSPECIFIED) {
      return { ok: true, value: readVersion(answer.version) }
    }
    return { ok: false, error: answer.error === TextErrorCode.MISSING ? 'missing' : 'invalid' }
  },
})
