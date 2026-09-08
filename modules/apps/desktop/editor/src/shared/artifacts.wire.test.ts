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
let asked: unknown[] = []

/** What the application answers with, in the words the schema writes it in. */
const answers = (said: unknown) => {
  asked = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (_url: string, init: { body: Uint8Array }) => {
      asked.push(JSON.parse(new TextDecoder().decode(init.body)))
      return new Response(JSON.stringify(said), {
        headers: { 'content-type': 'application/json' },
      })
    }),
  )
}

/** A build that does not carry the run at all refuses it by name. */
const refuses = (code: string) =>
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
    answers({
      artifacts: [
        { kind: 'ARTIFACT_KIND_TRANSCRIPT', state: 'STATE_DONE' },
        { kind: 'ARTIFACT_KIND_ARTICLE', state: 'STATE_RUNNING' },
      ],
    })

    expect(await running.carries('Talk.url')).toEqual({
      transcript: 'done',
      article: 'running',
    })
  })

  it('leaves out an artifact this window has no word for', async () => {
    answers({ artifacts: [{ kind: 99, state: 'STATE_DONE' }] })

    expect(await running.carries('Talk.url')).toEqual({})
  })

  it('is nothing made where the answer names no state', async () => {
    answers({ artifacts: [{ kind: 'ARTIFACT_KIND_OCR' }] })

    expect(await running.carries('Scan.pdf')).toEqual({ ocr: 'none' })
  })
})

describe('beginning a run', () => {
  it('names the artifact by the number the schema gave it', async () => {
    answers({ artifact: { state: 'STATE_QUEUED' } })

    expect(await running.makes('Scan.pdf', 'ocr')).toEqual({
      able: true,
      of: 'ocr',
      made: 'queued',
      error: '',
    })
    expect(asked[0]).toEqual({ path: 'Scan.pdf', kind: 'ARTIFACT_KIND_OCR' })
  })

  it('carries back what went wrong with it', async () => {
    answers({ artifact: { state: 'STATE_FAILED', error: 'the model is not here' } })

    expect(await running.makes('Scan.pdf', 'ocr')).toEqual({
      able: true,
      of: 'ocr',
      made: 'failed',
      error: 'the model is not here',
    })
  })

  it('is refused by a build that cannot do it at all', async () => {
    refuses('unimplemented')

    expect(await running.makes('Scan.pdf', 'ocr')).toEqual({ able: false })
  })

  it('throws where the run failed for any other reason', async () => {
    refuses('internal')

    await expect(running.makes('Scan.pdf', 'ocr')).rejects.toThrow()
  })
})

describe('putting a text right', () => {
  it('corrects the transcript of a file carrying one', async () => {
    answers({
      artifacts: [{ kind: 'ARTIFACT_KIND_TRANSCRIPT', state: 'STATE_DONE' }],
      artifact: { state: 'STATE_QUEUED' },
    })

    expect(await running.corrects('Talk.url')).toMatchObject({ of: 'transcript.corrected' })
  })

  it('corrects the reading of a file carrying no transcript', async () => {
    answers({ artifacts: [], artifact: { state: 'STATE_QUEUED' } })

    expect(await running.corrects('Scan.pdf')).toMatchObject({ of: 'ocr.corrected' })
  })
})

describe('asking an address afresh', () => {
  it('asks for the transcript of a file carrying one', async () => {
    answers({
      artifacts: [{ kind: 'ARTIFACT_KIND_TRANSCRIPT', state: 'STATE_DONE' }],
      artifact: { state: 'STATE_QUEUED' },
    })

    expect(await running.fetches('Talk.url')).toMatchObject({ of: 'transcript' })
  })

  it('asks for the prose of a file carrying no transcript', async () => {
    answers({ artifacts: [], artifact: { state: 'STATE_QUEUED' } })

    expect(await running.fetches('Site.url')).toMatchObject({ of: 'article' })
  })
})

describe('taking one away', () => {
  it('names the transcript by the number the schema gave it', async () => {
    answers({})

    expect(await running.deletesTranscript('Talk.url')).toBe(true)
    expect(asked[0]).toEqual({ path: 'Talk.url', kind: 'ARTIFACT_KIND_TRANSCRIPT' })
  })

  it('names the copy by the number the schema gave it', async () => {
    answers({})

    expect(await running.deletesCopy('Talk.url')).toBe(true)
    expect(asked[0]).toEqual({ path: 'Talk.url', kind: 'ARTIFACT_KIND_COPY' })
  })

  it('is refused by a build that cannot do it at all', async () => {
    refuses('unimplemented')

    expect(await running.deletesCopy('Talk.url')).toBe(false)
  })

  it('throws where it failed for any other reason', async () => {
    refuses('internal')

    await expect(running.deletesCopy('Talk.url')).rejects.toThrow()
  })
})
