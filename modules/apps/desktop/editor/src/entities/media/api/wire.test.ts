/**
 * Tests for the recordings wire client adapter.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('@/shared/clients', () => ({
  recordings: asked,
  transcripts: asked,
  articles: asked,
}))

vi.mock('@/entities/artifact/@x/media', () => ({
  running: {
    getArtifactStates: (path: string) => states(path),
  },
}))

/** What the file is said to carry, state by artifact. */
const states = vi.fn<(path: string) => Promise<Record<string, string>>>()

const asked = {
  getRecording: vi.fn(),
  readTranscript: vi.fn(),
  writeTranscript: vi.fn(),
  readArticle: vi.fn(),
}

const { recordings } = await import('./wire')

describe('recordings wire client', () => {
  beforeEach(() => {
    states.mockResolvedValue({ 'transcript.corrected': 'done' })
    for (const one of Object.values(asked)) one.mockReset()
  })

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

  it('asks for a recording by its path', async () => {
    asked.getRecording.mockResolvedValue({})

    await recordings.getSummary('audio.mp3')

    expect(asked.getRecording).toHaveBeenCalledWith({ path: 'audio.mp3' })
  })

  it('asks for a transcript by its path and carries no prose with it', async () => {
    asked.readTranscript.mockResolvedValue({ cues: [] })

    const transcript = await recordings.readTranscript('audio.mp3')

    expect(asked.readTranscript).toHaveBeenCalledWith({ path: 'audio.mp3' })
    expect(transcript.prose).toBe('')
  })

  it('asks for an article by its path and carries no cues with it', async () => {
    asked.readArticle.mockResolvedValue({ text: 'Article text' })

    const article = await recordings.readArticle('note.url')

    expect(asked.readArticle).toHaveBeenCalledWith({ path: 'note.url' })
    expect(article.cues).toEqual([])
  })

  it('writes the cues under the path', async () => {
    asked.writeTranscript.mockResolvedValue({})
    const cues = [{ text: 'Hello', from: 0, to: 1000 }]

    await recordings.writeTranscript('audio.mp3', cues)

    expect(asked.writeTranscript).toHaveBeenCalledWith({ path: 'audio.mp3', cues })
  })

  it('asks for the span of a cue under the path', async () => {
    asked.readTranscript.mockResolvedValue({ cues: [{ text: 'x', from: 5, to: 6 }] })

    await recordings.findCueTime('audio.mp3', { from: 1, to: 2 })

    expect(asked.readTranscript).toHaveBeenCalledWith({
      path: 'audio.mp3',
      span: { from: 1, to: 2 },
    })
  })

  it('finds no time where no cue holds the span', async () => {
    asked.readTranscript.mockResolvedValue({ cues: [] })

    expect(await recordings.findCueTime('audio.mp3', { from: 1, to: 2 })).toBeNull()
  })

  it('is not editable while a run puts the words right', async () => {
    states.mockResolvedValue({ 'transcript.corrected': 'running' })
    asked.readTranscript.mockResolvedValue({ cues: [] })
    asked.readArticle.mockResolvedValue({ text: '' })

    expect((await recordings.readTranscript('audio.mp3')).isEditable).toBe(false)
    expect((await recordings.readArticle('audio.mp3')).isEditable).toBe(false)
  })

  it('is editable where only another run is under way', async () => {
    states.mockResolvedValue({ transcript: 'running' })
    asked.readTranscript.mockResolvedValue({ cues: [] })

    expect((await recordings.readTranscript('audio.mp3')).isEditable).toBe(true)
  })
})
