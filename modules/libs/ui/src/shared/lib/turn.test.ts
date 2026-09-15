import { describe, expect, it } from 'vitest'
import { keyTurn } from './turn'

describe('what turns the page', () => {
  it('turns on and back on the keys anything read a page at a time is read with', () => {
    expect(keyTurn('ArrowRight')).toBe('next')
    expect(keyTurn('ArrowDown')).toBe('next')
    expect(keyTurn(' ')).toBe('next')
    expect(keyTurn('PageDown')).toBe('next')
    expect(keyTurn('ArrowLeft')).toBe('back')
    expect(keyTurn('ArrowUp')).toBe('back')
    expect(keyTurn('PageUp')).toBe('back')
    expect(keyTurn('Home')).toBe('first')
    expect(keyTurn('End')).toBe('last')
  })

  it('turns nothing on a key that is not one of them', () => {
    expect(keyTurn('a')).toBeUndefined()
    expect(keyTurn('Enter')).toBeUndefined()
    expect(keyTurn('Tab')).toBeUndefined()
  })
})
