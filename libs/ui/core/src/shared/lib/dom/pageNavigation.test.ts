import { describe, expect, it } from 'vitest'
import { getPageNavigationAction } from './pageNavigation'

describe('what triggers page navigation', () => {
  it('navigates next and back on standard navigation keys', () => {
    expect(getPageNavigationAction('ArrowRight')).toBe('next')
    expect(getPageNavigationAction('ArrowDown')).toBe('next')
    expect(getPageNavigationAction(' ')).toBe('next')
    expect(getPageNavigationAction('PageDown')).toBe('next')
    expect(getPageNavigationAction('ArrowLeft')).toBe('back')
    expect(getPageNavigationAction('ArrowUp')).toBe('back')
    expect(getPageNavigationAction('PageUp')).toBe('back')
    expect(getPageNavigationAction('Home')).toBe('first')
    expect(getPageNavigationAction('End')).toBe('last')
  })

  it('returns undefined on keys that are not navigation keys', () => {
    expect(getPageNavigationAction('a')).toBeUndefined()
    expect(getPageNavigationAction('Enter')).toBeUndefined()
    expect(getPageNavigationAction('Tab')).toBeUndefined()
  })
})
