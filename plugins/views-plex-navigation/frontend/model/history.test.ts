import { describe, expect, it } from 'vitest'
import { createBoardHistories } from './history'

describe('the history of a board', () => {
  it('has nothing to go back or forward to when empty', () => {
    const history = createBoardHistories()

    expect([history.canBack('a'), history.canForward('a')]).toStrictEqual([false, false])
    expect([history.back('a'), history.forward('a')]).toStrictEqual([null, null])
  })

  it('has no previous node after the first visit', () => {
    const history = createBoardHistories()
    history.visit('a', 'one.md')

    expect(history.canBack('a')).toBe(false)
  })

  it('goes back and forward in the order of the visits', () => {
    const history = createBoardHistories()
    ;['one.md', 'two.md', 'three.md'].forEach((node) => history.visit('a', node))

    expect([history.back('a'), history.back('a'), history.back('a')]).toStrictEqual([
      'two.md',
      'one.md',
      null,
    ])
    expect([history.forward('a'), history.forward('a'), history.forward('a')]).toStrictEqual([
      'two.md',
      'three.md',
      null,
    ])
  })

  it('discards the forward nodes when a new node is visited after going back', () => {
    const history = createBoardHistories()
    ;['one.md', 'two.md', 'three.md'].forEach((node) => history.visit('a', node))
    history.back('a')
    history.visit('a', 'four.md')

    expect(history.canForward('a')).toBe(false)
    expect(history.back('a')).toBe('two.md')
  })

  it('ignores a visit to the node already in front', () => {
    const history = createBoardHistories()
    history.visit('a', 'one.md')
    history.visit('a', 'one.md')

    expect(history.canBack('a')).toBe(false)
  })

  it('keeps the histories of two boards apart', () => {
    const history = createBoardHistories()
    history.visit('a', 'one.md')
    history.visit('a', 'two.md')
    history.visit('b', 'other.md')

    expect([history.canBack('a'), history.canBack('b')]).toStrictEqual([true, false])
  })

  it('starts again after the board is forgotten', () => {
    const history = createBoardHistories()
    history.visit('a', 'one.md')
    history.visit('a', 'two.md')
    history.forget('a')

    expect(history.canBack('a')).toBe(false)
  })
})
