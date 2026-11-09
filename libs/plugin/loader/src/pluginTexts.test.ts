import { describe, expect, it } from 'vitest'
import { TEXT_ERROR_CODE, type TextClient } from '@numen/core-client'
import { createPluginTexts } from './pluginTexts'

const version = {
  id: 'md.numen.engines.transcription/k',
  path: 'talks/morning.mp3',
  pluginId: 'md.numen.engines.transcription',
  key: 'k',
  sourceVersionId: '',
  kind: 'transcript',
  file: 'k.jsonl',
  createTimeUnixMs: 1_700_000_000_000n,
}

const createClient = (requests: unknown[]): TextClient => ({
  readLiveText: async () =>
    ({
      version,
      segments: [{ text: 'Good morning.\n', heading: '', selector: '{"t":[0,1.5]}' }],
    }) as unknown as Awaited<ReturnType<TextClient['readLiveText']>>,
  createTextEdit: async (request) => {
    requests.push(request)
    return {
      version: { ...version, pluginId: request.pluginId, sourceVersionId: request.sourceVersionId },
      error: TEXT_ERROR_CODE.UNSPECIFIED,
    } as unknown as Awaited<ReturnType<TextClient['createTextEdit']>>
  },
})

describe('the text versions a plugin view reads', () => {
  it('reads the live text with its selectors parsed and omits invalid selectors or empty headings', async () => {
    const readPaths: string[] = []
    const client: TextClient = {
      readLiveText: async (req) => {
        readPaths.push(req.path ?? '')
        return {
          version,
          segments: [
            { text: 'Good morning.\n', heading: '', selector: '{"t":[0,1.5]}' },
            { text: 'With heading.\n', heading: 'Intro', selector: '' },
            { text: 'Bad json.\n', heading: '', selector: '{not-json' },
            { text: 'Null selector.\n', heading: '', selector: 'null' },
            { text: 'Primitive selector.\n', heading: '', selector: '42' },
          ],
        } as unknown as Awaited<ReturnType<TextClient['readLiveText']>>
      },
      createTextEdit: async () => ({}) as never,
    }
    const live = await createPluginTexts(client, 'md.numen.views.media.viewer').readLiveText(
      'talks/morning.mp3',
    )

    expect(readPaths).toEqual(['talks/morning.mp3'])
    expect(live?.version.kind).toBe('transcript')
    expect(live?.segments).toStrictEqual([
      { text: 'Good morning.\n', selector: { t: [0, 1.5] } },
      { text: 'With heading.\n', heading: 'Intro' },
      { text: 'Bad json.\n' },
      { text: 'Null selector.\n' },
      { text: 'Primitive selector.\n' },
    ])
    expect('heading' in (live?.segments[0] ?? {})).toBe(false)
    expect('selector' in (live?.segments[1] ?? {})).toBe(false)
  })

  it('reads the note property a version was made from, and omits it for a version without one', async () => {
    const linked = {
      ...version,
      pluginId: 'md.numen.tools.media-importer',
      key: 'link',
      sourceProperty: { key: 'url', value: 'https://example.org/talk' },
    }
    const client: TextClient = {
      readLiveText: async (req) =>
        ({
          version: req.path === 'talks/link.md' ? linked : version,
          segments: [],
        }) as unknown as Awaited<ReturnType<TextClient['readLiveText']>>,
      createTextEdit: async () => ({}) as never,
    }
    const texts = createPluginTexts(client, 'md.numen.views.media-player')

    const fromProperty = await texts.readLiveText('talks/link.md')
    const fromFile = await texts.readLiveText('talks/morning.mp3')

    expect(fromProperty?.version.sourceProperty).toStrictEqual({
      key: 'url',
      value: 'https://example.org/talk',
    })
    expect('sourceProperty' in (fromFile?.version ?? {})).toBe(false)
  })

  it('returns null when readLiveText answers without a version', async () => {
    const client: TextClient = {
      readLiveText: async () =>
        ({ version: undefined, segments: [] }) as unknown as Awaited<
          ReturnType<TextClient['readLiveText']>
        >,
      createTextEdit: async () => ({}) as never,
    }
    const live = await createPluginTexts(client, 'md.numen.views.media.viewer').readLiveText(
      'talks/missing.mp3',
    )
    expect(live).toBeNull()
  })

  it('saves an edit in the folder of the plugin it was made for', async () => {
    const requests: unknown[] = []
    const texts = createPluginTexts(createClient(requests), 'md.numen.views.media.viewer')

    const edit = await texts.createEdit('talks/morning.mp3', version.id, [
      { text: 'Good morning!\n', selector: { t: [0, 1.5] } },
      { text: 'Second line.\n', heading: 'Part 2' },
    ])

    expect(edit.ok && edit.value.pluginId).toBe('md.numen.views.media.viewer')
    expect(requests).toEqual([
      {
        pluginId: 'md.numen.views.media.viewer',
        path: 'talks/morning.mp3',
        sourceVersionId: version.id,
        segments: [
          { text: 'Good morning!\n', heading: '', selector: '{"t":[0,1.5]}' },
          { text: 'Second line.\n', heading: 'Part 2', selector: '' },
        ],
      },
    ])
  })

  it('reports missing and invalid errors from createTextEdit', async () => {
    const makeErrorClient = (error: number, hasVersion = false): TextClient => ({
      readLiveText: async () => ({}) as never,
      createTextEdit: async () =>
        ({
          version: hasVersion ? version : undefined,
          error,
        }) as unknown as Awaited<ReturnType<TextClient['createTextEdit']>>,
    })

    const missing = await createPluginTexts(
      makeErrorClient(TEXT_ERROR_CODE.MISSING),
      'p',
    ).createEdit('a', 'v1', [])
    expect(missing).toEqual({ ok: false, error: 'missing' })

    const invalidWithVersion = await createPluginTexts(
      makeErrorClient(TEXT_ERROR_CODE.INVALID, true),
      'p',
    ).createEdit('a', 'v1', [])
    expect(invalidWithVersion).toEqual({ ok: false, error: 'invalid' })

    const unspecifiedWithoutVersion = await createPluginTexts(
      makeErrorClient(TEXT_ERROR_CODE.UNSPECIFIED, false),
      'p',
    ).createEdit('a', 'v1', [])
    expect(unspecifiedWithoutVersion).toEqual({ ok: false, error: 'invalid' })
  })
})
