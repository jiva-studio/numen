import { describe, expect, it } from 'vitest'
import { createMockVaultTexts } from '@numen/plugin-sdk/testing'
import { createTranscripts, findSegmentTime, getCues, getSegments } from './transcripts'

const transcript = {
  id: 'md.numen.engines.transcription/k',
  path: 'talks/river.mp3',
  pluginId: 'md.numen.engines.transcription',
  key: 'k',
  sourceVersionId: '',
  kind: 'transcript',
  file: 'k.jsonl',
  createTime: new Date(0),
}

const cues = [
  { text: 'The river is high.', from: 0, to: 1500 },
  { text: 'The heron waits.', from: 1500, to: 4000 },
]

describe('the transcript of a recording', () => {
  it('turns cues into segments and back', () => {
    expect(getSegments(cues)[1]).toEqual({ text: 'The heron waits.\n', selector: { t: [1.5, 4] } })
    expect(getCues(getSegments(cues))).toEqual(cues)
  })

  it('finds the moment the segment at a span begins', () => {
    const segments = getSegments(cues)
    expect(findSegmentTime(segments, { from: 20, to: 25 })).toBe(1500)
    expect(findSegmentTime(segments, { from: 500, to: 501 })).toBeNull()
  })

  it('reads the live transcript and saves an edit as a version made from it', async () => {
    const texts = createMockVaultTexts('md.numen.views.media-player')
    texts.setLive(transcript, getSegments(cues))
    const transcripts = createTranscripts(() => texts)

    expect(await transcripts.readTranscript('talks/river.mp3')).toEqual(cues)
    await transcripts.writeTranscript('talks/river.mp3', [
      cues[0]!,
      { ...cues[1]!, text: 'The heron waits still.' },
    ])

    expect(texts.edits.map((one) => one.version.sourceVersionId)).toEqual([transcript.id])
    expect((await transcripts.readTranscript('talks/river.mp3'))[1]?.text).toBe(
      'The heron waits still.',
    )
  })

  it('reads no cues where the file has no transcript', async () => {
    const transcripts = createTranscripts(() => createMockVaultTexts())
    expect(await transcripts.readTranscript('talks/river.mp3')).toEqual([])
    await expect(transcripts.writeTranscript('talks/river.mp3', cues)).rejects.toThrow()
  })
})
