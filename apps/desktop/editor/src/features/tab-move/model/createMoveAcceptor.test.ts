import { describe, expect, it, vi } from 'vitest'
import { TAB_MOVE_ERROR_CODE } from '@numen/core-client'
import { answerCore } from '@/testing/coreAnswers'
import { createMoveAcceptor } from './tabMove'

const answerAccept = (accept = vi.fn(() => ({ error: TAB_MOVE_ERROR_CODE.UNSPECIFIED }))) => {
  answerCore({ workspace: { acceptTabMove: accept } })
  return accept
}

describe('accepting a moved tab', () => {
  it('sends the accept for the window once what must come first has settled', async () => {
    const accept = answerAccept()
    let release: () => void = () => undefined
    const acceptor = createMoveAcceptor(
      'window-a',
      () => new Promise<void>((resolve) => (release = resolve)),
    )

    const accepted = acceptor('plex:2', 'move-1')
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(accept).not.toHaveBeenCalled()
    release()

    expect(await accepted).toStrictEqual({ ok: true, value: undefined })
    expect(accept).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ windowId: 'window-a', tabId: 'plex:2', moveId: 'move-1' }),
      expect.anything(),
    )
  })

  it('answers the error of a move the core refused', async () => {
    answerAccept(vi.fn(() => ({ error: TAB_MOVE_ERROR_CODE.TIMEOUT })))

    const result = await createMoveAcceptor('window-a', () => Promise.resolve())('plex:2', 'move-1')

    expect(result).toStrictEqual({ ok: false, error: 'timeout' })
  })
})
