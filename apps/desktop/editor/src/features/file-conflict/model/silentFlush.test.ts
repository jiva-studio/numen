/**
 * A plugin's vault_open sends a silent flush request: the page writes its buffers, reports the result
 * and shows no prompt. An outstanding conflict makes the plugin's call fail without a prompt to the user.
 */
import { describe, expect, it, vi } from 'vitest'

import { createFileFlush, type Conflict, type FlushResult } from './flush'

interface QuitMessage {
  token: string
  flush: boolean
  isSilent: boolean
}

/** What the application sends over the quit stream during tests. */
function createMockStream() {
  const events: QuitMessage[] = []
  let wake: (() => void) | null = null
  return {
    send(message: QuitMessage) {
      events.push(message)
      wake?.()
    },
    async *read() {
      for (;;) {
        while (events.length > 0) yield events.shift() as QuitMessage
        await new Promise<void>((awakened) => {
          wake = awakened
        })
      }
    },
  }
}

/** Lets the pending promise callbacks run. */
const settle = async () => {
  for (let i = 0; i < 20; i++) await new Promise((resolve) => setTimeout(resolve, 0))
}

function createPage() {
  const stream = createMockStream()
  const reported: { token: string; result?: FlushResult }[] = []
  const page = createFileFlush(
    {
      watchQuit: stream.read,
      reportFlush: async (token, result) => {
        reported.push({ token, result })
      },
    },
    vi.fn(() => Promise.resolve()),
  )
  page.start().catch((error: unknown) => {
    throw error
  })
  return { stream, reported, page }
}

const createConflict = (note: string): Conflict => ({
  note,
  keep: async () => {},
  take: async () => {},
})

describe('a silent flush request', () => {
  it('reports asking and shows no prompt while a conflict is outstanding', async () => {
    const { stream, reported, page } = createPage()
    page.raise(createConflict('Moraine.md'))
    await settle()

    stream.send({ token: '1', flush: true, isSilent: true })
    await settle()

    expect(reported).toEqual([{ token: '1', result: 'asking' }])
    expect(page.conflicts.value).toEqual([])
  })

  it('keeps a postponed conflict hidden', async () => {
    const { stream, reported, page } = createPage()
    page.raise(createConflict('Moraine.md'))
    stream.send({ token: '1', flush: true, isSilent: false })
    await settle()
    expect(page.conflicts.value.map((one) => one.note)).toEqual(['Moraine.md'])
    page.conflicts.value[0]?.later()
    expect(page.conflicts.value).toEqual([])

    stream.send({ token: '2', flush: true, isSilent: true })
    await settle()

    expect(reported).toContainEqual({ token: '2', result: 'asking' })
    expect(page.conflicts.value).toEqual([])
  })

  it('writes the buffers and reports written once they are written', async () => {
    const { stream, reported, page } = createPage()
    let finishWrite: (() => void) | null = null
    const handler = vi.fn(
      () =>
        new Promise<void>((resolve) => {
          finishWrite = resolve
        }),
    )
    page.addHandler(handler)

    stream.send({ token: '1', flush: true, isSilent: true })
    await settle()
    expect(handler).toHaveBeenCalledTimes(1)
    expect(reported).toEqual([])

    finishWrite?.()
    await settle()

    expect(reported).toEqual([{ token: '1', result: 'written' }])
  })
})

describe('a flush request of a close', () => {
  it('shows the outstanding conflict and reports asking', async () => {
    const { stream, reported, page } = createPage()
    page.raise(createConflict('Moraine.md'))
    await settle()

    stream.send({ token: '1', flush: true, isSilent: false })
    await settle()

    expect(reported).toEqual([{ token: '1', result: 'asking' }])
    expect(page.conflicts.value.map((one) => one.note)).toEqual(['Moraine.md'])
  })
})
