/**
 * Tests for the recordings wire client adapter.
 */
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/shared/clients', () => ({
  recordings: asked,
  transcripts: asked,
  articles: asked,
}))

vi.mock('@/shared/artifacts', () => ({
  running: {
    getArtifactStates: vi.fn().mockResolvedValue({ 'transcript.corrected': 'done' }),
  },
}))

const asked = {
  getRecording: vi.fn(),
  readTranscript: vi.fn(),
  writeTranscript: vi.fn(),
  readArticle: vi.fn(),
}

const { recordings } = await import('./wire')

describe('recordings wire client', () => {
  it('gets recording summary', async () => {
    asked.getRecording.mockResolvedValue({
      durationMs: 60000,
      mediaUrl: 'http://media/audio.mp3',
      mediaType: 'audio/mpeg',
      url: 'https://example.com/audio',
    })

    const summary = await recordings.getSummary('audio.mp3')
    expect(summary.duration).toBe(60000)
    expect(summary.mediaUrl).toBe('http://media/audio.mp3')
    expect(summary.mediaType).toBe('audio/mpeg')
    expect(summary.url).toBe('https://example.com/audio')
  })

  it('reads and writes transcript', async () => {
    asked.readTranscript.mockResolvedValue({
      cues: [{ text: 'Hello', from: 0, to: 1000 }],
    })

    const trans = await recordings.readTranscript('audio.mp3')
    expect(trans.cues.length).toBe(1)
    expect(trans.cues[0]?.text).toBe('Hello')
    expect(trans.isEditable).toBe(true)

    asked.writeTranscript.mockResolvedValue({})
    await recordings.writeTranscript('audio.mp3', [{ text: 'Hello', from: 0, to: 1000 }])
    expect(asked.writeTranscript).toHaveBeenCalled()
  })

  it('reads article and finds cue time', async () => {
    asked.readArticle.mockResolvedValue({ text: 'Article text' })
    const article = await recordings.readArticle('audio.mp3')
    expect(article.prose).toBe('Article text')
    expect(article.isEditable).toBe(true)

    asked.readTranscript.mockResolvedValue({
      cues: [{ text: 'Span cue', from: 500, to: 1500 }],
    })
    const time = await recordings.findCueTime('audio.mp3', { from: 0, to: 5 })
    expect(time).toBe(500)
  })

  it('gets task states', async () => {
    const states = await recordings.getTaskStates('audio.mp3')
    expect(states['transcript.corrected']).toBe('done')
  })
})
