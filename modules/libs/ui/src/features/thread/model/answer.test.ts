import { describe, expect, it, vi } from 'vitest'
import { createAnswer } from './answer'
import type { Turn } from '../lib/turn'

const create = () => {
  const put = vi.fn<(turn: Turn) => void>()
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
  return { answer: createAnswer({ put, drop, nextId }, paint), put, drop, frames, flush }
}

describe('an answer as it arrives', () => {
  it('puts the turn up with its first words at once, arriving', () => {
    const { answer, put, frames } = create()
    answer.say('Hel')
    expect(put).toHaveBeenCalledExactlyOnceWith({
      id: 'a1',
      voice: 'answered',
      text: 'Hel',
      state: 'arriving',
    })
    expect(frames).toHaveLength(0)
  })

  it('grows the turn once a frame, however many words came since the last', () => {
    const { answer, put, frames, flush } = create()
    answer.say('a')
    answer.say('b')
    answer.say('c')
    expect(frames).toHaveLength(1)
    flush()
    expect(put).toHaveBeenLastCalledWith({
      id: 'a1',
      voice: 'answered',
      text: 'abc',
      state: 'arriving',
    })
    expect(put).toHaveBeenCalledTimes(2)
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
    const { answer, put, flush } = create()
    answer.say('a')
    answer.say('b')
    answer.settle()
    put.mockClear()
    flush()
    expect(put).not.toHaveBeenCalled()
  })

  it('settles with its text and no state, and the next words begin another turn', () => {
    const { answer, put } = create()
    answer.say('done')
    answer.settle()
    expect(put).toHaveBeenLastCalledWith({ id: 'a1', voice: 'answered', text: 'done' })
    answer.say('next')
    expect(put).toHaveBeenLastCalledWith({
      id: 'a2',
      voice: 'answered',
      text: 'next',
      state: 'arriving',
    })
  })

  it('leaves no turn for an answer with nothing in it', () => {
    const { answer, put, drop } = create()
    answer.say('')
    put.mockClear()
    answer.settle()
    expect(drop).toHaveBeenCalledExactlyOnceWith('a1')
    expect(put).not.toHaveBeenCalled()
  })

  it('does nothing when settled with no answer open', () => {
    const { answer, put, drop } = create()
    answer.settle()
    expect(put).not.toHaveBeenCalled()
    expect(drop).not.toHaveBeenCalled()
  })

  it('marks the turn failed, and closes it', () => {
    const { answer, put } = create()
    answer.say('half')
    answer.fail()
    expect(put).toHaveBeenLastCalledWith({
      id: 'a1',
      voice: 'answered',
      text: 'half',
      state: 'failed',
    })
    expect(answer.isOpen()).toBe(false)
    answer.say('again')
    expect(put).toHaveBeenLastCalledWith({
      id: 'a2',
      voice: 'answered',
      text: 'again',
      state: 'arriving',
    })
  })

  it('drops a failed answer with nothing in it', () => {
    const { answer, put, drop } = create()
    answer.say('')
    put.mockClear()
    answer.fail()
    expect(drop).toHaveBeenCalledExactlyOnceWith('a1')
    expect(put).not.toHaveBeenCalled()
    expect(answer.isOpen()).toBe(false)
  })

  it('does nothing when failed with no answer open', () => {
    const { answer, put, drop } = create()
    answer.fail()
    expect(put).not.toHaveBeenCalled()
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
