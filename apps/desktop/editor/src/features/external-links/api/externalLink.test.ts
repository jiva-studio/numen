/**
 * A link that leads out of the application, pressed in a window with no
 * address bar.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { interceptExternalLinks } from './externalLink'

let stop: (() => void) | undefined

afterEach(() => {
  stop?.()
  stop = undefined
  document.body.innerHTML = ''
})

const pressLink = (href: string, inside = '', kind = 'click') => {
  const openLink = vi.fn()
  stop = interceptExternalLinks(openLink)
  document.body.innerHTML = `<a href="${href}">${inside || 'press'}</a>`
  const link = document.querySelector('a')!
  const target = inside === '' ? link : link.firstElementChild!
  const press = new MouseEvent(kind, { bubbles: true, cancelable: true })
  target.dispatchEvent(press)
  return { openLink, prevented: press.defaultPrevented }
}

describe('a press on a link', () => {
  it('takes the window nowhere and passes the address to a browser', () => {
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
    const { openLink, prevented } = pressLink('/pages/one')
    expect(prevented).toBe(false)
    expect(openLink).not.toHaveBeenCalled()
  })

  it('leaves a place inside the page alone', () => {
    const { openLink, prevented } = pressLink('#heading')
    expect(prevented).toBe(false)
    expect(openLink).not.toHaveBeenCalled()
  })

  it('leaves an href that is no address to the page', () => {
    const faults: unknown[] = []
    const onError = (event: ErrorEvent) => {
      faults.push(event.error)
    }
    window.addEventListener('error', onError)
    const { openLink, prevented } = pressLink('http://')
    window.removeEventListener('error', onError)
    expect(faults).toStrictEqual([])
    expect(prevented).toBe(false)
    expect(openLink).not.toHaveBeenCalled()
  })

  it.each(['mailto:someone@example.com', 'tel:+12345678'])('opens %s externally', (href) => {
    const { openLink, prevented } = pressLink(href)
    expect(prevented).toBe(true)
    expect(openLink).toHaveBeenCalledWith(href)
  })

  it('takes the window nowhere and passes no caller an address it cannot open', () => {
    for (const href of ['page://01J8', 'javascript:alert(1)', 'data:text/html,<h1>x']) {
      const { openLink, prevented } = pressLink(href)
      expect(prevented).toBe(true)
      expect(openLink).not.toHaveBeenCalled()
      stop?.()
    }
  })

  it('is left alone once the window is no longer retained', () => {
    const openLink = vi.fn()
    interceptExternalLinks(openLink)()
    document.body.innerHTML = '<a href="https://evil.example/">press</a>'
    const press = new MouseEvent('click', { bubbles: true, cancelable: true })
    document.querySelector('a')!.dispatchEvent(press)
    expect(press.defaultPrevented).toBe(false)
    expect(openLink).not.toHaveBeenCalled()
  })
})

describe('a press that reaches the retained window without a link', () => {
  /** The listener registered on the window, and the document it is registered on. */
  const hold = () => {
    const root = document.implementation.createHTMLDocument('')
    const add = vi.spyOn(root, 'addEventListener')
    const openLink = vi.fn<(href: string) => void>()
    stop = interceptExternalLinks(openLink, root)
    const listener = add.mock.calls[0]![1] as (press: MouseEvent) => void
    return { root, openLink, listener }
  }

  const press = (target: unknown) =>
    ({ target, preventDefault: vi.fn<() => void>() }) as unknown as MouseEvent

  it('is left alone when it has no target', () => {
    const { listener, openLink } = hold()
    const event = press(null)

    expect(() => listener(event)).not.toThrow()
    expect(event.preventDefault).not.toHaveBeenCalled()
    expect(openLink).not.toHaveBeenCalled()
  })

  it('is left alone when its target cannot hold a link', () => {
    const { listener, root, openLink } = hold()
    const event = press(root)

    expect(() => listener(event)).not.toThrow()
    expect(event.preventDefault).not.toHaveBeenCalled()
    expect(openLink).not.toHaveBeenCalled()
  })

  it('is left alone when it is not on a link', () => {
    const { listener, root, openLink } = hold()
    const event = press(root.createElement('p'))

    expect(() => listener(event)).not.toThrow()
    expect(event.preventDefault).not.toHaveBeenCalled()
    expect(openLink).not.toHaveBeenCalled()
  })
})

describe('a press that is answered before the page sees it', () => {
  it.each(['click', 'auxclick'])('takes the window nowhere from a %s the link stops', (kind) => {
    const openLink = vi.fn<(href: string) => void>()
    stop = interceptExternalLinks(openLink)
    document.body.innerHTML = '<a href="https://evil.example/">press</a>'
    const link = document.querySelector('a')!
    link.addEventListener(kind, (event) => event.stopPropagation())
    const press = new MouseEvent(kind, { bubbles: true, cancelable: true })

    link.dispatchEvent(press)

    expect(press.defaultPrevented).toBe(true)
    expect(openLink).toHaveBeenCalledWith('https://evil.example/')
  })

  it('is left alone from a middle button once the window is no longer retained', () => {
    const openLink = vi.fn<(href: string) => void>()
    interceptExternalLinks(openLink)()
    document.body.innerHTML = '<a href="https://evil.example/">press</a>'
    const press = new MouseEvent('auxclick', { bubbles: true, cancelable: true })
    document.querySelector('a')!.dispatchEvent(press)
    expect(press.defaultPrevented).toBe(false)
    expect(openLink).not.toHaveBeenCalled()
  })
})

describe('which addresses leave the window', () => {
  it.each(['protocol', 'hostname', 'port'] as const)('opens an address with another %s', (part) => {
    const target = new URL(document.location.href)
    if (part === 'protocol') target.protocol = target.protocol === 'https:' ? 'http:' : 'https:'
    if (part === 'hostname') target.hostname = 'other.example'
    if (part === 'port') target.port = target.port === '8080' ? '9090' : '8080'

    const { openLink, prevented } = pressLink(target.href)
    expect(prevented).toBe(true)
    expect(openLink).toHaveBeenCalledWith(target.href)
  })

  it('leaves a link to another path on the same host in the window', () => {
    const { openLink, prevented } = pressLink(new URL('/b?c#d', document.location.href).href)
    expect(prevented).toBe(false)
    expect(openLink).not.toHaveBeenCalled()
  })
})
