import { describe, expect, it, vi } from 'vitest'
import { createWorkLine } from './work'
import type { Turn } from '../lib/turn'
import type { SourceLocation } from '../lib/agent'

const PLACE: SourceLocation = { path: 'notes/a.md', span: { from: 2, to: 9 } }

const create = () => {
  const sendTurn = vi.fn<(turn: Turn) => void>()
  const drop = vi.fn<(id: string) => void>()
  const locations = new Map<string, SourceLocation>()
  const line = createWorkLine(
    { put: sendTurn, drop, locations, waitWords: 'Working' },
    'work',
    'wait',
  )
  return { line, put: sendTurn, drop, locations }
}

describe('the line that says what the agent has in hand', () => {
  it('says nothing on settling before anything was reached for', () => {
    const { line, put: sendTurn } = create()
    line.settle()
    expect(sendTurn).not.toHaveBeenCalled()
  })

  it('names the tool as a program is not named, and how much was written', () => {
    const { line, put: sendTurn } = create()
    line.reach({ tool: 'read_note', subject: 'Plans', written: 12 })
    expect(sendTurn).toHaveBeenCalledExactlyOnceWith({
      id: 'work',
      voice: 'doing',
      text: 'read note',
      subject: 'Plans',
      aside: '12 characters',
      state: 'arriving',
    })
  })

  it('settles in place once reached for', () => {
    const { line, put: sendTurn } = create()
    line.reach({ tool: 'read_note', subject: 'Plans', written: 0 })
    line.settle()
    expect(sendTurn).toHaveBeenLastCalledWith(
      expect.objectContaining({
        id: 'work',
        text: 'read note',
        subject: 'Plans',
        state: 'settled',
      }),
    )
  })

  it('does not put the line back by settling after it came down', () => {
    const { line, put: sendTurn, drop } = create()
    line.reach({ tool: 't', subject: 's', written: 0 })
    line.takeDown()
    sendTurn.mockClear()
    line.settle()
    expect(drop).toHaveBeenCalledWith('work')
    expect(sendTurn).not.toHaveBeenCalled()
  })

  it('can be opened while the call names a run of a source', () => {
    const { line, put: sendTurn, locations } = create()
    line.reach({ tool: 't', subject: 's', location: PLACE, written: 0 })
    expect(locations.get('work')).toBe(PLACE)
    expect(sendTurn).toHaveBeenLastCalledWith(expect.objectContaining({ canOpen: true }))
  })

  it('cannot be opened once a later call names no run', () => {
    const { line, put: sendTurn, locations } = create()
    line.reach({ tool: 't', subject: 's', location: PLACE, written: 0 })
    line.reach({
      tool: 't',
      subject: 's',
      location: { path: 'a', span: { from: 4, to: 4 } },
      written: 0,
    })
    expect(locations.has('work')).toBe(false)
    expect(sendTurn.mock.lastCall?.[0]).not.toHaveProperty('canOpen')
    line.reach({ tool: 't', subject: 's', location: PLACE, written: 0 })
    line.reach({ tool: 't', subject: 's', location: null, written: 0 })
    expect(locations.has('work')).toBe(false)
  })

  it('forgets the source when the line arrives down', () => {
    const { line, drop, locations } = create()
    line.reach({ tool: 't', subject: 's', location: PLACE, written: 0 })
    line.takeDown()
    expect(drop).toHaveBeenCalledWith('work')
    expect(locations.size).toBe(0)
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
    const { line, drop, locations } = create()
    line.reach({ tool: 't', subject: 's', location: PLACE, written: 0 })
    line.clear()
    expect(drop).toHaveBeenCalledWith('work')
    expect(drop).toHaveBeenCalledWith('wait')
    expect(locations.size).toBe(0)
  })
})
