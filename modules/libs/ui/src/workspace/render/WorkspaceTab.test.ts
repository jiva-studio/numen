/** What one tab says about itself: its name, what it carries, and its close. */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import WorkspaceTab from './WorkspaceTab.vue'

const mountTab = (props: Record<string, unknown> = {}, slots: Record<string, string> = {}) =>
  mount(WorkspaceTab, { props: { tab: 'notes', title: 'Notes', ...props }, slots })

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

describe('closing', () => {
  it('is asked for by the close', async () => {
    const tab = mountTab()
    await tab.find('button').trigger('click')

    expect(tab.emitted('close')).toHaveLength(1)
  })

  it('is not offered at all where the tab is not closable', () => {
    expect(mountTab({ closable: false }).find('button').exists()).toBe(false)
  })

  it('does not pick the tab up on the way', async () => {
    const tab = mountTab()
    await tab.find('button').trigger('pointerdown')

    expect(tab.emitted('lift')).toBeUndefined()
  })
})
