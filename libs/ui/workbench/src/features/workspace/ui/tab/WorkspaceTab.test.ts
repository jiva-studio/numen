import { enableAutoUnmount, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import WorkspaceTab from './WorkspaceTab.vue'

enableAutoUnmount(afterEach)

const LONG = 'A note with a name longer than the strip has room for'

const getTooltips = () => [...document.body.querySelectorAll<HTMLElement>('[role="tooltip"]')]

const mountTab = (props: Record<string, unknown> = {}, slots: Record<string, string> = {}) =>
  mount(WorkspaceTab, {
    props: { tab: 'notes', title: 'Notes', ...props },
    slots,
    attachTo: document.body,
  })

const pressOn = (tab: ReturnType<typeof mountTab>, button: number) => {
  const press = new PointerEvent('pointerdown', { button, bubbles: true, cancelable: true })
  tab.element.dispatchEvent(press)
  return press
}

describe('tab title', () => {
  it('renders the title text without a native title attribute or tooltip', async () => {
    const tab = mountTab({ title: LONG, isShowing: true })

    expect(tab.text()).toContain('A note with a name longer')
    expect(tab.element.querySelectorAll('[title]')).toHaveLength(0)
    expect(tab.attributes('title')).toBeUndefined()
    expect(getTooltips()).toHaveLength(0)

    await tab.trigger('mouseenter')
    expect(getTooltips()).toHaveLength(0)
    expect(tab.attributes('aria-describedby')).toBeUndefined()

    tab.element.focus()
    await tab.vm.$nextTick()
    expect(getTooltips()).toHaveLength(0)
  })

  it('sets the close button aria-label from the title', () => {
    expect(mountTab().find('button').attributes('aria-label')).toBe('Close Notes')
  })
})

describe('tab mark', () => {
  it('does not render a mark element when mark is undefined', () => {
    expect(mountTab().find('.tab__mark').exists()).toBe(false)
  })

  it('renders the default mark dot with aria-label and no native title attribute', () => {
    const mark = mountTab({ mark: 'unsaved' }).find('.tab__mark')

    expect(mark.exists()).toBe(true)
    expect(mark.attributes('aria-label')).toBe('unsaved')
    expect(mark.attributes('title')).toBeUndefined()
  })

  it('shows the mark in a tooltip while the pointer is over the mark', async () => {
    const tab = mountTab({ title: LONG, mark: 'unsaved' })
    const mark = tab.get('.tab__mark')

    await mark.trigger('mouseenter')

    const tooltip = getTooltips().find((one) => one.textContent?.trim() === 'unsaved')
    expect(tooltip).toBeDefined()
    expect(mark.attributes('aria-describedby')).toBe(tooltip?.id)
  })

  it('hides the mark tooltip when the pointer leaves the mark', async () => {
    const tab = mountTab({ mark: 'unsaved' })
    const mark = tab.get('.tab__mark')
    await mark.trigger('mouseenter')

    await mark.trigger('mouseleave')

    expect(getTooltips()).toHaveLength(0)
    expect(mark.attributes('aria-describedby')).toBeUndefined()
  })

  it('renders the custom mark slot when provided', () => {
    const tab = mountTab({ mark: 'stuck' }, { mark: '<b class="mine">{{ params.mark }}</b>' })

    expect(tab.find('.tab__mark').exists()).toBe(false)
    expect(tab.find('.mine').text()).toBe('stuck')
  })
})

describe('keyboard accessibility', () => {
  it('sets role="tab" and aria-selected', () => {
    const tab = mountTab({ isShowing: true })

    expect(tab.attributes('role')).toBe('tab')
    expect(tab.attributes('aria-selected')).toBe('true')
  })

  it('sets tabindex to 0 only when isShowing is true', () => {
    expect(mountTab({ isShowing: true }).attributes('tabindex')).toBe('0')
    expect(mountTab({ isShowing: false }).attributes('tabindex')).toBe('-1')
  })
})

describe('pointerdown on a tab', () => {
  it('emits press on primary button pointerdown', () => {
    const tab = mountTab()
    pressOn(tab, 0)

    expect(tab.emitted('press')).toHaveLength(1)
  })

  it('does not emit press on non-primary button pointerdown', () => {
    const tab = mountTab()
    pressOn(tab, 2)

    expect(tab.emitted('press')).toBeUndefined()
  })

  it('prevents default on primary button pointerdown', () => {
    const press = pressOn(mountTab(), 0)

    expect(press.defaultPrevented).toBe(true)
  })

  it('focuses the tab element on primary button pointerdown', () => {
    const tab = mountTab()
    pressOn(tab, 0)

    expect(document.activeElement).toBe(tab.element)
  })

  it('does not focus the tab on non-primary button pointerdown', () => {
    const tab = mountTab()
    pressOn(tab, 2)

    expect(document.activeElement).not.toBe(tab.element)
  })

  it('allows pointerdown to bubble to the parent pane', () => {
    const events: Event[] = []
    const listen = (event: Event) => events.push(event)
    document.addEventListener('pointerdown', listen)

    const press = pressOn(mountTab(), 0)
    document.removeEventListener('pointerdown', listen)

    expect(events).toContain(press)
  })

  it('does not prevent default on non-primary button pointerdown', () => {
    const press = pressOn(mountTab(), 2)

    expect(press.defaultPrevented).toBe(false)
  })
})

describe('contextmenu', () => {
  it('emits menu with pointer coordinates and prevents default', async () => {
    const tab = mountTab()
    const event = new MouseEvent('contextmenu', {
      bubbles: true,
      cancelable: true,
      clientX: 12,
      clientY: 34,
    })
    tab.element.dispatchEvent(event)

    expect(tab.emitted('menu')).toStrictEqual([[{ x: 12, y: 34 }]])
    expect(event.defaultPrevented).toBe(true)
  })
})

describe('close button', () => {
  it('emits close on click', async () => {
    const tab = mountTab()
    await tab.find('button').trigger('click')

    expect(tab.emitted('close')).toHaveLength(1)
  })

  it('does not emit press on close button pointerdown', async () => {
    const tab = mountTab()
    await tab.find('button').trigger('pointerdown')

    expect(tab.emitted('press')).toBeUndefined()
  })
})
