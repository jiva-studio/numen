/**
 * What the vault last reported about each deck file, stored under its path and
 * kept across reads.
 */
import { describe, expect, it } from 'vitest'
import { WORDS, type DeckProblem } from '../../cards'
import { createDeckAnswers } from './answers'

const PROBLEM: DeckProblem = { kind: 'unknown', card: 0, template: null, field: '', text: 'Odd' }

const readItem = (title: string | null) => ({ problems: [PROBLEM], bound: 0, title })

describe('the title of a deck file', () => {
  it('is the name of the file until the vault reads one', () => {
    const answers = createDeckAnswers()

    expect(answers.getTitle('cards/Animals.md')).toBe('Animals.md')
  })

  it('is the title a read came back with', () => {
    const answers = createDeckAnswers()

    answers.recordRead('Animals.md', readItem('Beasts'))

    expect(answers.getTitle('Animals.md')).toBe('Beasts')
  })

  it('remains through a read that reached no title', () => {
    const answers = createDeckAnswers()
    answers.recordRead('Animals.md', readItem('Beasts'))

    answers.recordRead('Animals.md', readItem(null))

    expect(answers.getTitle('Animals.md')).toBe('Beasts')
  })

  it('is the title the window was told before any read answered', () => {
    const answers = createDeckAnswers()

    answers.setTitle('Animals.md', 'Told')

    expect(answers.getTitle('Animals.md')).toBe('Told')
  })
})

describe('what is forgotten of a file', () => {
  it('is its problems, its errors and its title', () => {
    const answers = createDeckAnswers()
    answers.recordRead('Animals.md', { ...readItem('Beasts'), error: 'notADeck' })

    answers.forgetFile('Animals.md')

    expect(answers.getProblems('Animals.md')).toStrictEqual([])
    expect(answers.getErrorMessage('Animals.md', true)).toBe(WORDS.unreachable)
    expect(answers.getTitle('Animals.md')).toBe('Animals.md')
  })

  it('is nothing of another file', () => {
    const answers = createDeckAnswers()
    answers.recordRead('Other.md', readItem('Other title'))

    answers.forgetFile('Animals.md')

    expect(answers.getProblems('Other.md')).toStrictEqual([PROBLEM])
    expect(answers.getTitle('Other.md')).toBe('Other title')
  })
})

describe('a file that moved', () => {
  it('carries its problems and its title to where it now remains', () => {
    const answers = createDeckAnswers()
    answers.recordRead('Old.md', readItem('Beasts'))

    answers.moveFile('Old.md', 'New.md')

    expect(answers.getProblems('New.md')).toStrictEqual([PROBLEM])
    expect(answers.getTitle('New.md')).toBe('Beasts')
  })

  it('leaves nothing at the name it left', () => {
    const answers = createDeckAnswers()
    answers.recordRead('Old.md', readItem('Beasts'))

    answers.moveFile('Old.md', 'New.md')

    expect(answers.getProblems('Old.md')).toStrictEqual([])
    expect(answers.getTitle('Old.md')).toBe('Old.md')
  })

  it('leaves what remains at the name it went to, where the old name retained nothing', () => {
    const answers = createDeckAnswers()
    answers.recordRead('New.md', readItem('Beasts'))

    answers.moveFile('Missing.md', 'New.md')

    expect(answers.getProblems('New.md')).toStrictEqual([PROBLEM])
    expect(answers.getTitle('New.md')).toBe('Beasts')
  })

  it('carries a title that was told, where no read answered', () => {
    const answers = createDeckAnswers()
    answers.setTitle('Old.md', 'Told')

    answers.moveFile('Old.md', 'New.md')

    expect(answers.getTitle('New.md')).toBe('Told')
  })
})
