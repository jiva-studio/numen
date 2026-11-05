/** The command that opens a note, over the line a heading or a passage stands for. */
import { describe, expect, it, vi } from 'vitest'
import { COMMAND } from '@/shared/commandIds'
import { createInvocation } from '../../lib/offers/invocation'
import { ANSWER_WORDS } from '../../words'
import type { CommandTarget } from '../../types'
import type { CommandDeps } from './deps'
import { runInvocation } from './handlers'

const target = (override: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: '',
  kind: null,
  path: 'reef/Coral.md',
  title: 'Coral',
  file: 'reef/Coral.md',
  vault: { id: 'v', name: 'V' },
  isReady: true,
  ...override,
})

const open = async (chosen: CommandTarget) => {
  const openFile = vi.fn()
  await runInvocation(
    createInvocation(COMMAND.openNote, chosen),
    { notes: { openFile }, writeMessage: vi.fn() } as unknown as CommandDeps,
    ANSWER_WORDS,
  )
  return openFile
}

describe('the invocation of a command over a target with a line', () => {
  it('keeps the line', () => {
    expect(createInvocation(COMMAND.openNote, target({ line: 7 })).line).toBe(7)
    expect(createInvocation(COMMAND.openNote, target({ line: 0 })).line).toBe(0)
  })

  it('has no line where the target has none', () => {
    expect(createInvocation(COMMAND.openNote, target()).line).toBeUndefined()
  })
})

describe('the command that opens a note', () => {
  it('opens the note in the current tab at the line of the target', async () => {
    const openFile = await open(target({ line: 7 }))

    expect(openFile).toHaveBeenCalledExactlyOnceWith('reef/Coral.md', 'Coral', 'current', {
      line: 7,
    })
  })

  it('opens the note at its first line where the line is zero', async () => {
    const openFile = await open(target({ line: 0 }))

    expect(openFile.mock.calls[0]?.[3]).toStrictEqual({ line: 0 })
  })

  it('opens the note from the start where the target has no line', async () => {
    const openFile = await open(target())

    expect(openFile).toHaveBeenCalledTimes(1)
    expect(openFile.mock.calls[0]?.slice(0, 3)).toEqual(['reef/Coral.md', 'Coral', 'current'])
    expect(openFile.mock.calls[0]?.[3]).toBeUndefined()
  })
})
