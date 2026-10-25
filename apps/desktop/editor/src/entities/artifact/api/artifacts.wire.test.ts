/**
 * The runs a model makes over a file, read by the client the schema generates.
 *
 * The window names an artifact by a word of its own, and the schema names it by
 * a number. What is proved here is that the two meet: a word goes out as the
 * number the schema gave it, and a number comes back as the word.
 */
import { describe, expect, it, vi } from 'vitest'

vi.stubGlobal('window', { location: { origin: 'http://numen.invalid' } })

/** What the window asked for, as the transport wrote it out. */
let request: unknown[] = []

/** What the application answers with, in the words the schema writes it in. */
const answerWith = (answer: unknown) => {
  request = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (_url: string, init: { body: Uint8Array }) => {
      request.push(JSON.parse(new TextDecoder().decode(init.body)))
      return new Response(JSON.stringify(answer), {
        headers: { 'content-type': 'application/json' },
      })
    }),
  )
}

/** A build that does not carry the run at all refuses it by name. */
const stubError = (code: string) =>
  vi.stubGlobal(
    'fetch',
    vi.fn(
      async () =>
        new Response(JSON.stringify({ code, message: 'no' }), {
          status: code === 'unimplemented' ? 501 : 500,
          headers: { 'content-type': 'application/json' },
        }),
    ),
  )

const { running } = await import('./artifacts')

describe('what a file carries', () => {
  it('is keyed by the word the window uses for each artifact', async () => {
    answerWith({
      artifacts: [
        { kind: 'ARTIFACT_KIND_TRANSCRIPT', state: 'STATE_DONE' },
        { kind: 'ARTIFACT_KIND_ARTICLE', state: 'STATE_RUNNING' },
      ],
    })

    expect(await running.getArtifactStates('Talk.url')).toEqual({
      transcript: 'done',
      article: 'running',
    })
  })

  it('asks for the artifacts of the path', async () => {
    answerWith({ artifacts: [] })

    await running.getArtifactStates('Talk.url')

    expect(request[0]).toEqual({ path: 'Talk.url' })
  })

  it('leaves out an artifact this window has no word for', async () => {
    answerWith({ artifacts: [{ kind: 99, state: 'STATE_DONE' }] })

    expect(await running.getArtifactStates('Talk.url')).toEqual({})
  })

  it('is nothing made where the answer names no state', async () => {
    answerWith({ artifacts: [{ kind: 'ARTIFACT_KIND_TRANSCRIPT' }] })

    expect(await running.getArtifactStates('Talk.mp3')).toEqual({ transcript: 'none' })
  })

  it('is nothing made where the answer names a state this window has no word for', async () => {
    answerWith({ artifacts: [{ kind: 'ARTIFACT_KIND_TRANSCRIPT', state: 99 }] })

    expect(await running.getArtifactStates('Talk.mp3')).toEqual({ transcript: 'none' })
  })
})

describe('beginning a run', () => {
  it('names the artifact by the number the schema gave it', async () => {
    answerWith({ artifact: { state: 'STATE_QUEUED' } })

    expect(await running.createArtifact('Talk.mp3', 'transcript')).toEqual({
      able: true,
      of: 'transcript',
      made: 'queued',
      error: '',
    })
    expect(request[0]).toEqual({ path: 'Talk.mp3', kind: 'ARTIFACT_KIND_TRANSCRIPT' })
  })

  it('is nothing made, and no error, where the answer names no artifact', async () => {
    answerWith({})

    expect(await running.createArtifact('Talk.mp3', 'transcript')).toEqual({
      able: true,
      of: 'transcript',
      made: 'none',
      error: '',
    })
  })

  it('carries back what went wrong with it', async () => {
    answerWith({ artifact: { state: 'STATE_FAILED', error: 'the model is not here' } })

    expect(await running.createArtifact('Talk.mp3', 'transcript')).toEqual({
      able: true,
      of: 'transcript',
      made: 'failed',
      error: 'the model is not here',
    })
  })

  it('fails by a build that cannot do it at all', async () => {
    stubError('unimplemented')

    expect(await running.createArtifact('Talk.mp3', 'transcript')).toEqual({ able: false })
  })

  it('throws where the run failed for any other reason', async () => {
    stubError('internal')

    await expect(running.createArtifact('Talk.mp3', 'transcript')).rejects.toThrow()
  })
})

describe('asking an address afresh', () => {
  it('asks for the transcript of a file carrying one', async () => {
    answerWith({
      artifacts: [{ kind: 'ARTIFACT_KIND_TRANSCRIPT', state: 'STATE_DONE' }],
      artifact: { state: 'STATE_QUEUED' },
    })

    expect(await running.fetchArtifact('Talk.url')).toMatchObject({ of: 'transcript' })
  })

  it('asks for the prose of a file carrying no transcript', async () => {
    answerWith({ artifacts: [], artifact: { state: 'STATE_QUEUED' } })

    expect(await running.fetchArtifact('Site.url')).toMatchObject({ of: 'article' })
  })
})

describe('taking one away', () => {
  it('names the transcript by the number the schema gave it', async () => {
    answerWith({})

    expect(await running.deleteTranscript('Talk.url')).toBe(true)
    expect(request[0]).toEqual({ path: 'Talk.url', kind: 'ARTIFACT_KIND_TRANSCRIPT' })
  })

  it('names the copy by the number the schema gave it', async () => {
    answerWith({})

    expect(await running.deleteCopy('Talk.url')).toBe(true)
    expect(request[0]).toEqual({ path: 'Talk.url', kind: 'ARTIFACT_KIND_COPY' })
  })

  it('fails by a build that cannot do it at all', async () => {
    stubError('unimplemented')

    expect(await running.deleteCopy('Talk.url')).toBe(false)
  })

  it('throws where it failed for any other reason', async () => {
    stubError('internal')

    await expect(running.deleteCopy('Talk.url')).rejects.toThrow()
  })
})
