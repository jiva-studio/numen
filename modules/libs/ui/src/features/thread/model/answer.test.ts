import { describe, expect, it, vi } from 'vitest'
import { createAnswer } from './answer'
import type { Turn } from '../lib/turn'

const create = () => {
  const sendTurn = vi.fn<(turn: Turn) => void>()
  const drop = vi.fn<(id: string) => void>()
  let count = 0
  const nextId = () => `a${++count}`
  const frames: Array<() => void> = []
  const paint = (draw: () => void) => {
    frames.push(draw)
  }
  const flush = () => {
    frames.splice(0).forEach((draw) => {
      draw()
    })
  }
  return { answer: createAnswer({ put: sendTurn, drop, nextId }, paint), put: sendTurn, drop, frames, flush }
}

describe('an answer as it arrives', () => {
  it('puts the turn up with its first words at once, arriving', () => {
    const { answer, put: sendTurn, frames } = create()
    answer.say('Hel')
    expect(sendTurn).toHaveBeenCalledExactlyOnceWith({
      id: 'a1',
      voice: 'answered',
      text: 'Hel',
      state: 'arriving',
    })
    expect(frames).toHaveLength(0)
  })

  it('grows the turn once a frame, however many words came since the last', () => {
    const { answer, put: sendTurn, frames, flush } = create()
    answer.say('a')
    answer.say('b')
    answer.say('c')
    expect(frames).toHaveLength(1)
    flush()
    expect(sendTurn).toHaveBeenLastCalledWith({
      id: 'a1',
      voice: 'answered',
      text: 'abc',
      state: 'arriving',
    })
    expect(sendTurn).toHaveBeenCalledTimes(2)
  })

  it('asks for another frame for words that arrive after one was drawn', () => {
    const { answer, frames, flush } = create()
    answer.say('a')
    answer.say('b')
    flush()
    answer.say('c')
    expect(frames).toHaveLength(1)
  })

  it('draws nothing from a frame that arrives after the answer ended', () => {
    const { answer, put: sendTurn, flush } = create()
    answer.say('a')
    answer.say('b')
    answer.settle()
    sendTurn.mockClear()
    flush()
    expect(sendTurn).not.toHaveBeenCalled()
  })

  it('settles with its text and no state, and the next words begin another turn', () => {
    const { answer, put: sendTurn } = create()
    answer.say('done')
    answer.settle()
    expect(sendTurn).toHaveBeenLastCalledWith({ id: 'a1', voice: 'answered', text: 'done' })
    answer.say('next')
    expect(sendTurn).toHaveBeenLastCalledWith({
      id: 'a2',
      voice: 'answered',
      text: 'next',
      state: 'arriving',
    })
  })

  it('leaves no turn for an answer with nothing in it', () => {
    const { answer, put: sendTurn, drop } = create()
    answer.say('')
    sendTurn.mockClear()
    answer.settle()
    expect(drop).toHaveBeenCalledExactlyOnceWith('a1')
    expect(sendTurn).not.toHaveBeenCalled()
  })

  it('does nothing when settled with no answer open', () => {
    const { answer, put: sendTurn, drop } = create()
    answer.settle()
    expect(sendTurn).not.toHaveBeenCalled()
    expect(drop).not.toHaveBeenCalled()
  })

  it('marks the turn failed, and closes it', () => {
    const { answer, put: sendTurn } = create()
    answer.say('half')
    answer.fail()
    expect(sendTurn).toHaveBeenLastCalledWith({
      id: 'a1',
      voice: 'answered',
      text: 'half',
      state: 'failed',
    })
    expect(answer.isOpen()).toBe(false)
    answer.say('again')
    expect(sendTurn).toHaveBeenLastCalledWith({
      id: 'a2',
      voice: 'answered',
      text: 'again',
      state: 'arriving',
    })
  })

  it('drops a failed answer with nothing in it', () => {
    const { answer, put: sendTurn, drop } = create()
    answer.say('')
    sendTurn.mockClear()
    answer.fail()
    expect(drop).toHaveBeenCalledExactlyOnceWith('a1')
    expect(sendTurn).not.toHaveBeenCalled()
    expect(answer.isOpen()).toBe(false)
  })

  it('does nothing when failed with no answer open', () => {
    const { answer, put: sendTurn, drop } = create()
    answer.fail()
    expect(sendTurn).not.toHaveBeenCalled()
    expect(drop).not.toHaveBeenCalled()
  })

  it('says whether it is open and whether it has words', () => {
    const { answer } = create()
    expect(answer.isOpen()).toBe(false)
    expect(answer.hasContent()).toBe(false)
    expect(answer.getText()).toBe('')
    answer.say('hi')
    expect(answer.isOpen()).toBe(true)
    expect(answer.hasContent()).toBe(true)
    expect(answer.getText()).toBe('hi')
    answer.settle()
    expect(answer.isOpen()).toBe(false)
  })
})
