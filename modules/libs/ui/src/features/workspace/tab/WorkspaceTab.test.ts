/** What one tab says about itself: its name, what it carries, and its close. */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import WorkspaceTab from './WorkspaceTab.vue'

const mountTab = (props: Record<string, unknown> = {}, slots: Record<string, string> = {}) =>
  mount(WorkspaceTab, {
    props: { tab: 'notes', title: 'Notes', ...props },
    slots,
    attachTo: document.body,
  })

/** A press on the tab itself, handed back for what became of it. */
const pressOn = (tab: ReturnType<typeof mountTab>, button: number) => {
  const press = new PointerEvent('pointerdown', { button, bubbles: true, cancelable: true })
  tab.element.dispatchEvent(press)
  return press
}

describe('what a tab is called', () => {
  it('is drawn, and carried whole for a name too long to draw', () => {
    const tab = mountTab({ title: 'A note with a name longer than the strip has room for' })

    expect(tab.text()).toContain('A note with a name longer')
    expect(tab.find('span[title]').attributes('title')).toBe(
      'A note with a name longer than the strip has room for',
    )
  })

  it('is what the close is announced by', () => {
    expect(mountTab().find('button').attributes('aria-label')).toBe('Close Notes')
  })
})

describe('what a tab carries', () => {
  it('is nothing until it is given something', () => {
    expect(mountTab().find('.tab__mark').exists()).toBe(false)
  })

  it('is drawn as a dot, and said in the word it was given', () => {
    const mark = mountTab({ mark: 'unsaved' }).find('.tab__mark')

    expect(mark.exists()).toBe(true)
    expect(mark.attributes('aria-label')).toBe('unsaved')
    expect(mark.attributes('title')).toBe('unsaved')
  })

  it('is drawn by the caller where the caller draws one', () => {
    const tab = mountTab({ mark: 'stuck' }, { mark: '<b class="mine">{{ params.mark }}</b>' })

    expect(tab.find('.tab__mark').exists()).toBe(false)
    expect(tab.find('.mine').text()).toBe('stuck')
  })
})

describe('what a tab is to a keyboard', () => {
  it('is a tab in a strip, and says whether it is the one showing', () => {
    const tab = mountTab({ showing: true })

    expect(tab.attributes('role')).toBe('tab')
    expect(tab.attributes('aria-selected')).toBe('true')
  })

  it('is stopped at only where it is the one showing', () => {
    expect(mountTab({ showing: true }).attributes('tabindex')).toBe('0')
    expect(mountTab({ showing: false }).attributes('tabindex')).toBe('-1')
  })
})

describe('picking a tab up', () => {
  it('is asked for under the primary button', () => {
    const tab = mountTab()
    pressOn(tab, 0)

    expect(tab.emitted('lift')).toHaveLength(1)
  })

  it('is not asked for under any other button', () => {
    const tab = mountTab()
    pressOn(tab, 2)

    expect(tab.emitted('lift')).toBeUndefined()
  })

  it('selects no text as it travels', () => {
    const press = pressOn(mountTab(), 0)

    expect(press.defaultPrevented).toBe(true)
  })

  it('takes the keyboard, which is where the strip is walked on from', () => {
    const tab = mountTab()
    pressOn(tab, 0)

    expect(document.activeElement).toBe(tab.element)
  })

  it('leaves the keyboard where it was under any other button', () => {
    const tab = mountTab()
    pressOn(tab, 2)

    expect(document.activeElement).not.toBe(tab.element)
  })

  it('leaves the press to the pane under it', () => {
    const heard: Event[] = []
    const listen = (event: Event) => heard.push(event)
    document.addEventListener('pointerdown', listen)

    const press = pressOn(mountTab(), 0)
    document.removeEventListener('pointerdown', listen)

    expect(heard).toContain(press)
  })

  it('lets a press of any other button be answered elsewhere', () => {
    const press = pressOn(mountTab(), 2)

    expect(press.defaultPrevented).toBe(false)
  })
})

describe('closing', () => {
  it('is asked for by the close', async () => {
    const tab = mountTab()
    await tab.find('button').trigger('click')

    expect(tab.emitted('close')).toHaveLength(1)
  })

  it('does not pick the tab up on the way', async () => {
    const tab = mountTab()
    await tab.find('button').trigger('pointerdown')

    expect(tab.emitted('lift')).toBeUndefined()
  })
})
