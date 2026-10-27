import { describe, expect, it } from 'vitest'
import { TextErrorCode } from '@numen/protocol'
import { createPluginTexts, type TextClient } from './pluginTexts'

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
      error: TextErrorCode.UNSPECIFIED,
    } as unknown as Awaited<ReturnType<TextClient['createTextEdit']>>
  },
})

describe('the text versions a plugin view reads', () => {
  it('reads the live text with its selectors parsed', async () => {
    const live = await createPluginTexts(
      createClient([]),
      'md.numen.views.media-player',
    ).readLiveText('talks/morning.mp3')

    expect(live?.version.kind).toBe('transcript')
    expect(live?.segments).toEqual([{ text: 'Good morning.\n', selector: { t: [0, 1.5] } }])
  })

  it('saves an edit in the folder of the plugin it was made for', async () => {
    const requests: unknown[] = []
    const texts = createPluginTexts(createClient(requests), 'md.numen.views.media-player')

    const edit = await texts.createEdit('talks/morning.mp3', version.id, [
      { text: 'Good morning!\n', selector: { t: [0, 1.5] } },
    ])

    expect(edit.ok && edit.value.pluginId).toBe('md.numen.views.media-player')
    expect(requests).toEqual([
      {
        pluginId: 'md.numen.views.media-player',
        path: 'talks/morning.mp3',
        sourceVersionId: version.id,
        segments: [{ text: 'Good morning!\n', heading: '', selector: '{"t":[0,1.5]}' }],
      },
    ])
  })
})
