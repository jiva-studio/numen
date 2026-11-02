/**
 * The pictures a drawn document waits for before its columns are read.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { PICTURES_WAIT, findPending, whenArrived } from './pictures'

const picture = (isComplete: boolean): HTMLImageElement => {
  const one = document.createElement('img')
  Object.defineProperty(one, 'complete', { value: isComplete })
  return one
}

const textWith = (...pictures: HTMLImageElement[]): HTMLElement => {
  const text = document.createElement('div')
  text.append(...pictures)
  return text
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('the pictures that have not arrived', () => {
  it('are the ones not complete', () => {
    const notArrived = picture(false)

    expect(findPending(textWith(picture(true), notArrived))).toStrictEqual([notArrived])
  })

  it('are none in a text without pictures', () => {
    expect(findPending(textWith())).toStrictEqual([])
  })
})

describe('waiting for pictures', () => {
  it('runs once all of them have arrived or failed', () => {
    const [one, other] = [picture(false), picture(false)]
    const run = vi.fn()
    whenArrived([one, other], run)

    one.dispatchEvent(new Event('load'))
    expect(run).not.toHaveBeenCalled()
    other.dispatchEvent(new Event('error'))

    expect(run).toHaveBeenCalledTimes(1)
  })

  it('runs after the limit when a picture does not answer, and once', () => {
    const one = picture(false)
    const run = vi.fn()
    whenArrived([one], run)

    vi.advanceTimersByTime(PICTURES_WAIT)
    one.dispatchEvent(new Event('load'))

    expect(run).toHaveBeenCalledTimes(1)
  })
})
