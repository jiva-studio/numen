/**
 * The info button of a setting row: what it opens, and where the focus goes.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import SettingRow from './SettingRow.vue'

const DETAILS =
  'Lets the agent read the hooks and skills of your own setup. Off, it runs with the tools of the vault only.'

const mountRow = (details?: string) =>
  mount(SettingRow, {
    attachTo: document.body,
    props: {
      at: 'agent-hooks',
      name: 'Hooks & skills',
      detail: 'Use your own hooks and skills',
      ...(details === undefined ? {} : { details }),
    },
  })

const getInfoButton = () => {
  const button = document.querySelector<HTMLButtonElement>('[aria-label="About Hooks & skills"]')
  if (!button) throw new Error('no info button')
  return button
}

const getPopover = () => document.querySelector<HTMLElement>('[role="dialog"]')

afterEach(() => {
  document.body.innerHTML = ''
})

describe('SettingRow', () => {
  it('shows the short description and no info button for a setting without details', () => {
    const row = mountRow()
    expect(row.text()).toContain('Use your own hooks and skills')
    expect(document.querySelector('[aria-label="About Hooks & skills"]')).toBeNull()
  })

  it('opens the details from the info button and moves the focus into them', async () => {
    mountRow(DETAILS)
    const button = getInfoButton()
    expect(button.getAttribute('aria-expanded')).toBe('false')

    button.click()
    await new Promise((resolve) => setTimeout(resolve))

    const popover = getPopover()
    expect(popover?.getAttribute('aria-label')).toBe('About Hooks & skills')
    expect(popover?.textContent).toContain(DETAILS)
    expect(button.getAttribute('aria-expanded')).toBe('true')
    expect(document.activeElement).toBe(popover)
  })

  it('closes the details on Escape and gives the focus back to the info button', async () => {
    mountRow(DETAILS)
    getInfoButton().click()
    await new Promise((resolve) => setTimeout(resolve))

    getPopover()?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await new Promise((resolve) => setTimeout(resolve))

    expect(getPopover()).toBeNull()
    expect(document.activeElement).toBe(getInfoButton())
  })

  it('closes the details on a press outside the row', async () => {
    mountRow(DETAILS)
    getInfoButton().click()
    await new Promise((resolve) => setTimeout(resolve))

    document.body.dispatchEvent(new Event('pointerdown', { bubbles: true }))
    await new Promise((resolve) => setTimeout(resolve))

    expect(getPopover()).toBeNull()
  })

  it('closes the details on a second press of the info button', async () => {
    mountRow(DETAILS)
    getInfoButton().click()
    await new Promise((resolve) => setTimeout(resolve))

    getInfoButton().click()
    await new Promise((resolve) => setTimeout(resolve))

    expect(getPopover()).toBeNull()
  })
})
