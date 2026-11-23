import { describe, expect, it, vi } from 'vitest'
import { createWorkLine } from './work'
import type { Turn } from '../../types'

const create = () => {
  const sendTurn = vi.fn<(turn: Turn) => void>()
  const drop = vi.fn<(id: string) => void>()
  const paths = new Map<string, string>()
  let next = 0
  const nextId = () => `call-${next++}`
  const line = createWorkLine({ put: sendTurn, drop, paths, waitWords: 'Working', createId: nextId }, 'wait')
  return { line, put: sendTurn, drop, paths }
}

describe('the lines that say what the agent has in hand', () => {
  it('ends no call that was never reached for', () => {
    const { line, put: sendTurn } = create()
    line.end('c1')
    expect(sendTurn).not.toHaveBeenCalled()
  })

  it('names the tool as a program is not named, with what the call is about', () => {
    const { line, put: sendTurn } = create()
    line.reach('c1', { title: 'read_note', subject: 'Plans', path: '' })
    expect(sendTurn).toHaveBeenCalledExactlyOnceWith({
      id: 'call-0',
      voice: 'doing',
      text: 'read note',
      subject: 'Plans',
      state: 'arriving',
    })
  })

  it('settles its own line when a call ends', () => {
    const { line, put: sendTurn } = create()
    line.reach('c1', { title: 'Read note', subject: 'Plans', path: '' })
    line.reach('c2', { title: 'Search notes', subject: '', path: '' })
    line.end('c1')
    expect(sendTurn).toHaveBeenLastCalledWith(
      expect.objectContaining({ id: 'call-0', text: 'Read note', state: 'settled' }),
    )
  })

  it('keeps a line for each call', () => {
    const { line, put: sendTurn, drop } = create()
    line.reach('c1', { title: 'a', subject: '', path: '' })
    line.end('c1')
    line.reach('c2', { title: 'b', subject: '', path: '' })
    expect(sendTurn.mock.calls.map(([turn]) => turn.id)).toEqual(['call-0', 'call-0', 'call-1'])
    expect(drop).not.toHaveBeenCalled()
  })

  it('can be opened while the call works on a file', () => {
    const { line, put: sendTurn, paths } = create()
    line.reach('c1', { title: 't', subject: 's', path: 'notes/Orchard.md' })
    expect(paths.get('call-0')).toBe('notes/Orchard.md')
    expect(sendTurn).toHaveBeenLastCalledWith(expect.objectContaining({ canOpen: true }))
  })

  it('keeps the file of an earlier call when a later call works on no file', () => {
    const { line, paths } = create()
    line.reach('c1', { title: 't', subject: 's', path: 'notes/Orchard.md' })
    line.reach('c2', { title: 't', subject: 's', path: '' })
    expect(paths.get('call-0')).toBe('notes/Orchard.md')
    expect(paths.has('call-1')).toBe(false)
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

  it('settles the calls in hand and takes the wait down on clear', () => {
    const { line, put: sendTurn, drop } = create()
    line.reach('c1', { title: 'Read note', subject: 's', path: 'notes/Orchard.md' })
    line.clear()
    expect(drop).toHaveBeenCalledExactlyOnceWith('wait')
    expect(sendTurn).toHaveBeenLastCalledWith(
      expect.objectContaining({ id: 'call-0', state: 'settled', canOpen: true }),
    )
  })

  it('says how many characters the call has written until it ends', () => {
    const { line, put: sendTurn } = create()
    line.reach('c1', { title: 'Write note', subject: 'Pear trees', path: '' })
    line.write('c1', 12015)
    expect(sendTurn).toHaveBeenLastCalledWith(
      expect.objectContaining({
        text: 'Write note',
        aside: '12 015 characters',
        state: 'arriving',
      }),
    )
    line.end('c1')
    expect(sendTurn.mock.lastCall?.[0]).not.toHaveProperty('aside')
  })

  it('writes nothing for a call that is not shown', () => {
    const { line, put: sendTurn } = create()
    line.write('c1', 40)
    expect(sendTurn).not.toHaveBeenCalled()
  })
})
