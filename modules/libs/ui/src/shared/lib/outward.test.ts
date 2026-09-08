/**
 * A link that leads out of the application, pressed in a window with no
 * address bar.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { holdsTheWindow } from './outward'

let stop: (() => void) | undefined

afterEach(() => {
  stop?.()
  stop = undefined
  document.body.innerHTML = ''
})

/** One link on the page, and what happened when it was pressed. */
const pressing = (href: string, inside = '', kind = 'click') => {
  const opens = vi.fn()
  stop = holdsTheWindow(opens)
  document.body.innerHTML = `<a href="${href}">${inside || 'press'}</a>`
  const link = document.querySelector('a')!
  const target = inside === '' ? link : link.firstElementChild!
  const press = new MouseEvent(kind, { bubbles: true, cancelable: true })
  target.dispatchEvent(press)
  return { opens, prevented: press.defaultPrevented }
}

describe('a press on a link', () => {
  it('takes the window nowhere and hands the address to a browser', () => {
    const { opens, prevented } = pressing('https://evil.example/')
    expect(prevented).toBe(true)
    expect(opens).toHaveBeenCalledWith('https://evil.example/')
  })

  it('is answered the same from a middle button', () => {
    const { opens, prevented } = pressing('https://evil.example/', '', 'auxclick')
    expect(prevented).toBe(true)
    expect(opens).toHaveBeenCalledWith('https://evil.example/')
  })

  it('is answered from whatever inside the link was pressed', () => {
    const { opens, prevented } = pressing('https://evil.example/', '<strong>press</strong>')
    expect(prevented).toBe(true)
    expect(opens).toHaveBeenCalledWith('https://evil.example/')
  })

  it('leaves a page this window serves alone', () => {
    const { opens, prevented } = pressing('/notes/one')
    expect(prevented).toBe(false)
    expect(opens).not.toHaveBeenCalled()
  })

  it('leaves a place inside the page alone', () => {
    const { opens, prevented } = pressing('#heading')
    expect(prevented).toBe(false)
    expect(opens).not.toHaveBeenCalled()
  })

  it('hands a letter to whatever writes one', () => {
    const { opens, prevented } = pressing('mailto:someone@example.com')
    expect(prevented).toBe(true)
    expect(opens).toHaveBeenCalledWith('mailto:someone@example.com')
  })

  it('takes the window nowhere and hands nobody an address it cannot open', () => {
    for (const href of ['note://01J8', 'javascript:alert(1)', 'data:text/html,<h1>x']) {
      const { opens, prevented } = pressing(href)
      expect(prevented).toBe(true)
      expect(opens).not.toHaveBeenCalled()
      stop?.()
    }
  })

  it('is left alone once the window is no longer held', () => {
    const opens = vi.fn()
    holdsTheWindow(opens)()
    document.body.innerHTML = '<a href="https://evil.example/">press</a>'
    const press = new MouseEvent('click', { bubbles: true, cancelable: true })
    document.querySelector('a')!.dispatchEvent(press)
    expect(press.defaultPrevented).toBe(false)
    expect(opens).not.toHaveBeenCalled()
  })
})
