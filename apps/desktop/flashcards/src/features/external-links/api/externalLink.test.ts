import { expect, it, vi } from 'vitest'
import { interceptExternalLinks } from './externalLink'

it.each([
  ['https://other.example/review', true, true],
  ['mailto:deck@example.org', true, true],
  ['tel:+12345678', true, true],
  ['javascript:void(0)', true, false],
  ['#front', false, false],
] as const)('handles a card link to %s', (href, isPrevented, isOpened) => {
  const open = vi.fn()
  const stop = interceptExternalLinks(open)
  const anchor = document.createElement('a')
  const label = document.createElement('span')
  anchor.href = href
  anchor.append(label)
  document.body.append(anchor)
  try {
    const event = new MouseEvent('click', { bubbles: true, cancelable: true })
    label.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(isPrevented)
    expect(open.mock.calls).toEqual(isOpened ? [[href]] : [])
  } finally {
    stop()
    anchor.remove()
  }
})
