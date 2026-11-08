/** The command that opens a file that is no note, in the reader its type has, at a place in it. */
import { describe, expect, it, vi } from 'vitest'
import { COMMAND } from '@/shared/commandIds'
import { createInvocation } from '../../lib/entries/invocation'
import { ANSWER_WORDS } from '../../words'
import type { CommandTarget } from '../../types'
import type { CommandDeps } from './deps'
import { runInvocation } from './handlers'

const createTarget = (override: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: '',
  kind: null,
  path: '',
  title: 'Atlas',
  file: 'library/a.bin',
  vault: { id: 'v', name: 'V' },
  isReady: true,
  ...override,
})

const open = async (target: CommandTarget) => {
  const openFile = vi.fn()
  const emitEvent = vi.fn()
  const writeMessage = vi.fn()
  await runInvocation(
    createInvocation(COMMAND.openFile, target),
    { notes: { openFile }, emitEvent, writeMessage } as unknown as CommandDeps,
    ANSWER_WORDS,
  )
  return { openFile, emitEvent, writeMessage }
}

describe('the invocation of a command over a target with a selector', () => {
  it('keeps the selector', () => {
    expect(
      createInvocation(COMMAND.openFile, createTarget({ selector: { page: 3 } })).selector,
    ).toEqual({
      page: 3,
    })
  })

  it('has no selector where the target has none', () => {
    expect(createInvocation(COMMAND.openFile, createTarget()).selector).toBeUndefined()
  })
})

describe('the command that opens a document', () => {
  it('opens the file of the target in the current tab at the selector', async () => {
    const { openFile } = await open(createTarget({ selector: { page: 3 } }))

    expect(openFile).toHaveBeenCalledExactlyOnceWith('library/a.bin', 'Atlas', 'current', {
      selector: { page: 3 },
    })
  })

  it('opens the file from its start where the target has no selector', async () => {
    const { openFile } = await open(createTarget())

    expect(openFile).toHaveBeenCalledTimes(1)
    expect(openFile.mock.calls[0]?.slice(0, 3)).toEqual(['library/a.bin', 'Atlas', 'current'])
    expect(openFile.mock.calls[0]?.[3]?.selector).toBeUndefined()
  })

  it('opens the selector of a recording as it is', async () => {
    const { openFile } = await open(createTarget({ selector: { t: [12, 14] } }))

    expect(openFile.mock.calls[0]?.[3]).toEqual({ selector: { t: [12, 14] } })
  })

  it('names the tab by the path of the file where the file has no title', async () => {
    const { openFile } = await open(createTarget({ title: '' }))

    expect(openFile.mock.calls[0]?.slice(0, 3)).toEqual([
      'library/a.bin',
      'library/a.bin',
      'current',
    ])
  })

  it('emits the event of the command and writes no message', async () => {
    const { emitEvent, writeMessage } = await open(createTarget({ selector: { page: 3 } }))

    expect(emitEvent).toHaveBeenCalledExactlyOnceWith('onCommand:app:file.open')
    expect(writeMessage).toHaveBeenCalledExactlyOnceWith('')
  })
})
