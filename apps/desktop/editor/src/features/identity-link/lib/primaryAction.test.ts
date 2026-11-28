import { describe, expect, it } from 'vitest'
import { getPrimaryAction } from './primaryAction'

const IDLE = { isBusy: false, hasInput: false, isPasting: false }

describe('the primary action of a link dialog', () => {
  it('has none while the link is scanned and not pasted', () => {
    expect(getPrimaryAction('scan', IDLE)).toBeUndefined()
  })

  it('uses a pasted link once the field has text', () => {
    expect(getPrimaryAction('scan', { ...IDLE, isPasting: true })).toStrictEqual({
      label: 'Use link',
      isDisabled: true,
    })
    expect(getPrimaryAction('scan', { ...IDLE, isPasting: true, hasInput: true })?.isDisabled).toBe(
      false,
    )
  })

  it('lets the host approve the new device unless a call is running', () => {
    expect(getPrimaryAction('peer', IDLE)).toStrictEqual({ label: 'Link device', isDisabled: false })
    expect(getPrimaryAction('peer', { ...IDLE, isBusy: true })?.isDisabled).toBe(true)
  })

  it('tries again from a stalled, refused or failed link', () => {
    for (const view of ['stalled', 'refused', 'failed'] as const) {
      expect(getPrimaryAction(view, IDLE)?.label).toBe('Try again')
    }
  })

  it('has none where the dialog only waits or is done', () => {
    for (const view of ['preparing', 'qr', 'waiting', 'linking', 'linked'] as const) {
      expect(getPrimaryAction(view, IDLE)).toBeUndefined()
    }
  })
})
