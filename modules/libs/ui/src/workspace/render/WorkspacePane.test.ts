/** What a pane does with a keyboard, and what it tells a screen reader. */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import WorkspacePane from './WorkspacePane.vue'
import { pane } from '../model'

const three = () => pane('main', ['plex', 'chat', 'notes'], 'chat')

const mountPane = (props: Record<string, unknown> = {}, slots: Record<string, string> = {}) =>
  mount(WorkspacePane, {
    attachTo: document.body,
    props: {
      pane: three(),
      titles: { plex: 'Plex', chat: 'Chat', notes: 'Notes' },
      ...props,
    },
    slots,
  })

const strip = (held: ReturnType<typeof mountPane>) => held.findAll('[data-workspace-tab]')

const panels = (held: ReturnType<typeof mountPane>) => held.findAll('[role="tabpanel"]')

const named = () => document.activeElement?.getAttribute('data-workspace-tab')

describe('what the strip is', () => {
  it('is a list of tabs, each standing over a panel', () => {
    const held = mountPane()

    expect(held.find('[role="tablist"]').exists()).toBe(true)
    expect(strip(held)).toHaveLength(3)
    expect(panels(held)).toHaveLength(3)

    for (const [at, tab] of strip(held).entries()) {
      expect(tab.attributes('aria-controls')).toBe(panels(held)[at]?.attributes('id'))
      expect(panels(held)[at]?.attributes('aria-labelledby')).toBe(tab.attributes('id'))
    }
  })

  it('says which of them is showing', () => {
    const selected = strip(mountPane()).map((tab) => tab.attributes('aria-selected'))
    expect(selected).toStrictEqual(['false', 'true', 'false'])
  })

  it('is stopped at once, at the tab that is showing', () => {
    const stops = strip(mountPane()).map((tab) => tab.attributes('tabindex'))
    expect(stops).toStrictEqual(['-1', '0', '-1'])
  })

  it('hands each tab what that tab is carrying', () => {
    const held = mountPane({ marks: { chat: 'unsaved' } })
    const marks = strip(held).map((tab) => {
      const drawn = tab.find('.tab__mark')
      return drawn.exists() ? drawn.attributes('aria-label') : null
    })

    expect(marks).toStrictEqual([null, 'unsaved', null])
  })
})

describe('walking the strip', () => {
  it('steps along with the arrows, and shows what it reaches', async () => {
    const held = mountPane()
    await strip(held)[1]?.trigger('keydown', { key: 'ArrowRight' })

    expect(held.emitted('choose')).toStrictEqual([['notes']])
    expect(named()).toBe('notes')
  })

  it('meets its own ends', async () => {
    const held = mountPane({ pane: pane('main', ['plex', 'chat', 'notes'], 'plex') })
    await strip(held)[0]?.trigger('keydown', { key: 'ArrowLeft' })

    expect(held.emitted('choose')).toStrictEqual([['notes']])
    expect(named()).toBe('notes')
  })

  it('goes to either end by Home and End', async () => {
    const held = mountPane()
    await strip(held)[1]?.trigger('keydown', { key: 'Home' })
    await strip(held)[0]?.trigger('keydown', { key: 'End' })

    expect(held.emitted('choose')).toStrictEqual([['plex'], ['notes']])
  })

  it('leaves every other key to whatever else wants it', async () => {
    const held = mountPane()
    await strip(held)[1]?.trigger('keydown', { key: 'ArrowDown' })
    await strip(held)[1]?.trigger('keydown', { key: 'a' })

    expect(held.emitted('choose')).toBeUndefined()
  })
})

describe('the way out of a panel', () => {
  const inside = '<button class="inside">inside</button>'

  it('is Escape, which lands on the tab the panel is held under', async () => {
    const held = mountPane({}, { tab: inside })
    const first = held.findAll<HTMLButtonElement>('.inside')[1]

    first?.element.focus()
    await first?.trigger('keydown', { key: 'Escape' })

    expect(named()).toBe('chat')
  })

  it('leaves Escape alone where the panel has already acted on it', async () => {
    const held = mountPane({}, { tab: inside })
    const first = held.findAll<HTMLButtonElement>('.inside')[1]

    first?.element.addEventListener('keydown', (event) => event.preventDefault())
    first?.element.focus()
    await first?.trigger('keydown', { key: 'Escape' })

    expect(document.activeElement).toBe(first?.element)
  })
})
