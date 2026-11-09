/** The browser ports: what they read off the page, and what they answer without one. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  getCurrentDate,
  getNow,
  hasDocument,
  hasWindow,
  measureElementRect,
  measureRect,
  measureWindow,
} from './dom'

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('the window', () => {
  it('is measured at its inner size', () => {
    const was = { width: window.innerWidth, height: window.innerHeight }
    Object.defineProperty(window, 'innerWidth', { value: 1024, configurable: true })
    Object.defineProperty(window, 'innerHeight', { value: 768, configurable: true })
    try {
      expect(measureWindow()).toEqual({ width: 1024, height: 768 })
    } finally {
      Object.defineProperty(window, 'innerWidth', { value: was.width, configurable: true })
      Object.defineProperty(window, 'innerHeight', { value: was.height, configurable: true })
    }
  })

  it('is there, with its document, in a browser', () => {
    expect(hasWindow()).toBe(true)
    expect(hasDocument()).toBe(true)
  })

  it('answers 800 by 600 where there is no window or document', () => {
    vi.stubGlobal('window', undefined)
    vi.stubGlobal('document', undefined)
    expect(hasWindow()).toBe(false)
    expect(hasDocument()).toBe(false)
    expect(measureWindow()).toEqual({ width: 800, height: 600 })
  })
})

describe('the measurements of an element', () => {
  it('are the rectangle the browser reports for it', () => {
    const element = document.createElement('div')
    const rect = new DOMRect(4, 8, 120, 30)
    vi.spyOn(element, 'getBoundingClientRect').mockReturnValue(rect)
    expect(measureElementRect(element)).toBe(rect)
    expect(measureRect(element)).toBe(rect)
  })

  it('are none where there is no element', () => {
    expect(measureRect(null)).toBeUndefined()
    expect(measureRect(undefined)).toBeUndefined()
  })
})

describe('the clock', () => {
  it('is the wall clock, which a test controls with fake timers', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-04T05:06:07.000Z'))
    expect(getNow()).toBe(Date.parse('2026-03-04T05:06:07.000Z'))
    expect(getCurrentDate().toISOString()).toBe('2026-03-04T05:06:07.000Z')
  })
})
