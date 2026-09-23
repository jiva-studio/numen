/**
 * A link that leads out of the application, pressed in a window with no
 * address bar.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { holdWindow } from './outward'

let stop: (() => void) | undefined

afterEach(() => {
  stop?.()
  stop = undefined
  document.body.innerHTML = ''
})

/** One link on the page, and what happened when it was pressed. */
const pressLink = (href: string, inside = '', kind = 'click') => {
  const openLink = vi.fn()
  stop = holdWindow(openLink)
  document.body.innerHTML = `<a href="${href}">${inside || 'press'}</a>`
  const link = document.querySelector('a')!
  const target = inside === '' ? link : link.firstElementChild!
  const press = new MouseEvent(kind, { bubbles: true, cancelable: true })
  target.dispatchEvent(press)
  return { openLink, prevented: press.defaultPrevented }
}

describe('a press on a link', () => {
  it('takes the window nowhere and hands the address to a browser', () => {
    const { openLink, prevented } = pressLink('https://evil.example/')
    expect(prevented).toBe(true)
    expect(openLink).toHaveBeenCalledWith('https://evil.example/')
  })

  it('is answered the same from a middle button', () => {
    const { openLink, prevented } = pressLink('https://evil.example/', '', 'auxclick')
    expect(prevented).toBe(true)
    expect(openLink).toHaveBeenCalledWith('https://evil.example/')
  })

  it('is answered from whatever inside the link was pressed', () => {
    const { openLink, prevented } = pressLink('https://evil.example/', '<strong>press</strong>')
    expect(prevented).toBe(true)
    expect(openLink).toHaveBeenCalledWith('https://evil.example/')
  })

  it('leaves a page this window serves alone', () => {
    const { openLink, prevented } = pressLink('/notes/one')
    expect(prevented).toBe(false)
    expect(openLink).not.toHaveBeenCalled()
  })

  it('leaves a place inside the page alone', () => {
    const { openLink, prevented } = pressLink('#heading')
    expect(prevented).toBe(false)
    expect(openLink).not.toHaveBeenCalled()
  })

  it('hands a letter to whatever writes one', () => {
    const { openLink, prevented } = pressLink('mailto:someone@example.com')
    expect(prevented).toBe(true)
    expect(openLink).toHaveBeenCalledWith('mailto:someone@example.com')
  })

  it('takes the window nowhere and hands nobody an address it cannot open', () => {
    for (const href of ['note://01J8', 'javascript:alert(1)', 'data:text/html,<h1>x']) {
      const { openLink, prevented } = pressLink(href)
      expect(prevented).toBe(true)
      expect(openLink).not.toHaveBeenCalled()
      stop?.()
    }
  })

  it('is left alone once the window is no longer held', () => {
    const openLink = vi.fn()
    holdWindow(openLink)()
    document.body.innerHTML = '<a href="https://evil.example/">press</a>'
    const press = new MouseEvent('click', { bubbles: true, cancelable: true })
    document.querySelector('a')!.dispatchEvent(press)
    expect(press.defaultPrevented).toBe(false)
    expect(openLink).not.toHaveBeenCalled()
  })
})
