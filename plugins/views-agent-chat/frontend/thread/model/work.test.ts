import { describe, expect, it, vi } from 'vitest'
import { createWorkLine } from './work'
import type { Turn } from '../../types'

const create = () => {
  const sendTurn = vi.fn<(turn: Turn) => void>()
  const drop = vi.fn<(id: string) => void>()
  const paths = new Map<string, string>()
  const line = createWorkLine({ put: sendTurn, drop, paths, waitWords: 'Working' }, 'work', 'wait')
  return { line, put: sendTurn, drop, paths }
}

describe('the line that says what the agent has in hand', () => {
  it('says nothing on settling before anything was reached for', () => {
    const { line, put: sendTurn } = create()
    line.settle()
    expect(sendTurn).not.toHaveBeenCalled()
  })

  it('names the tool as a program is not named, with what the call is about', () => {
    const { line, put: sendTurn } = create()
    line.reach({ title: 'read_note', subject: 'Plans', path: '' })
    expect(sendTurn).toHaveBeenCalledExactlyOnceWith({
      id: 'work',
      voice: 'doing',
      text: 'read note',
      subject: 'Plans',
      state: 'arriving',
    })
  })

  it('settles in place once reached for', () => {
    const { line, put: sendTurn } = create()
    line.reach({ title: 'Read note', subject: 'Plans', path: '' })
    line.settle()
    expect(sendTurn).toHaveBeenLastCalledWith(
      expect.objectContaining({
        id: 'work',
        text: 'Read note',
        subject: 'Plans',
        state: 'settled',
      }),
    )
  })

  it('does not put the line back by settling after it came down', () => {
    const { line, put: sendTurn, drop } = create()
    line.reach({ title: 't', subject: 's', path: '' })
    line.dropWorkLine()
    sendTurn.mockClear()
    line.settle()
    expect(drop).toHaveBeenCalledWith('work')
    expect(sendTurn).not.toHaveBeenCalled()
  })

  it('can be opened while the call works on a file', () => {
    const { line, put: sendTurn, paths } = create()
    line.reach({ title: 't', subject: 's', path: 'notes/Orchard.md' })
    expect(paths.get('work')).toBe('notes/Orchard.md')
    expect(sendTurn).toHaveBeenLastCalledWith(expect.objectContaining({ canOpen: true }))
  })

  it('cannot be opened once a later call works on no file', () => {
    const { line, put: sendTurn, paths } = create()
    line.reach({ title: 't', subject: 's', path: 'notes/Orchard.md' })
    line.reach({ title: 't', subject: 's', path: '' })
    expect(paths.has('work')).toBe(false)
    expect(sendTurn.mock.lastCall?.[0]).not.toHaveProperty('canOpen')
  })

  it('clears the file path when the line is removed', () => {
    const { line, drop, paths } = create()
    line.reach({ title: 't', subject: 's', path: 'notes/Orchard.md' })
    line.dropWorkLine()
    expect(drop).toHaveBeenCalledWith('work')
    expect(paths.size).toBe(0)
  })

  it('puts the wait up and takes it down', () => {
    const { line, put: sendTurn, drop } = create()
    line.setWaiting(true)
    expect(sendTurn).toHaveBeenCalledExactlyOnceWith({
      id: 'wait',
      voice: 'doing',
      text: 'Working',
      subject: '',
      state: 'arriving',
    })
    line.setWaiting(false)
    expect(drop).toHaveBeenCalledWith('wait')
  })

  it('brings both lines down on clear', () => {
    const { line, drop, paths } = create()
    line.reach({ title: 't', subject: 's', path: 'notes/Orchard.md' })
    line.clear()
    expect(drop).toHaveBeenCalledWith('work')
    expect(drop).toHaveBeenCalledWith('wait')
    expect(paths.size).toBe(0)
  })
})
