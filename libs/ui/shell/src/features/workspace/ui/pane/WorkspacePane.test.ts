/** What a pane does with a keyboard, and what it tells a screen reader. */
import { mount } from '@vue/test-utils'
import { computed } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import WorkspacePane from './WorkspacePane.vue'
import { WORKSPACE_CONTEXT, type WorkspaceContext } from '../../model/context'
import { pane, type Tab } from '../../lib/node'

const three = () => pane('main', ['plex', 'chat', 'notes'], 'chat')

const TITLES: Readonly<Record<string, string>> = { plex: 'Plex', chat: 'Chat', notes: 'Notes' }

const createContext = (marks: Readonly<Record<string, string>> = {}): WorkspaceContext => ({
  findTab: (id): Tab | undefined =>
    TITLES[id] === undefined
      ? undefined
      : { id, title: TITLES[id], ...(marks[id] === undefined ? {} : { mark: marks[id] }) },
  focus: 'main',
  minimum: 220,
  choose: vi.fn(),
  close: vi.fn(),
  lift: vi.fn(),
  menu: vi.fn(),
  claim: vi.fn(),
  resize: vi.fn(),
  show: vi.fn(),
})

const mountPane = (props: Record<string, unknown> = {}, slots: Record<string, string> = {}) => {
  const { marks, ...rest } = props as { marks?: Readonly<Record<string, string>> }
  return mount(WorkspacePane, {
    attachTo: document.body,
    global: { provide: { [WORKSPACE_CONTEXT as symbol]: computed(() => createContext(marks)) } },
    props: { pane: three(), ...rest },
    slots,
  })
}

const strip = (wrapper: ReturnType<typeof mountPane>) => wrapper.findAll('[data-workspace-tab]')

const panels = (wrapper: ReturnType<typeof mountPane>) => wrapper.findAll('[role="tabpanel"]')

const getFocusedTab = () => document.activeElement?.getAttribute('data-workspace-tab')

describe('what the strip is', () => {
  it('is a list of tabs, each standing over a panel', () => {
    const target = mountPane()

    expect(target.find('[role="tablist"]').exists()).toBe(true)
    expect(strip(target)).toHaveLength(3)
    expect(panels(target)).toHaveLength(3)

    for (const [at, tab] of strip(target).entries()) {
      expect(tab.attributes('aria-controls')).toBe(panels(target)[at]?.attributes('id'))
      expect(panels(target)[at]?.attributes('aria-labelledby')).toBe(tab.attributes('id'))
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

  it('passes each tab what that tab is carrying', () => {
    const target = mountPane({ marks: { chat: 'unsaved' } })
    const marks = strip(target).map((tab) => {
      const renderedMark = tab.find('.tab__mark')
      return renderedMark.exists() ? renderedMark.attributes('aria-label') : null
    })

    expect(marks).toStrictEqual([null, 'unsaved', null])
  })

  it('is drawn where the pane holds tabs', () => {
    expect(mountPane().find('[data-workspace-strip]').exists()).toBe(true)
  })

  it('is not drawn at all where the pane holds none', () => {
    expect(
      mountPane({ pane: pane('main', []) })
        .find('[data-workspace-strip]')
        .exists(),
    ).toBe(false)
  })
})

describe('a pane holding nothing', () => {
  const mountEmptyPane = (slots: Record<string, string> = {}) =>
    mountPane({ pane: pane('main', []) }, slots)

  it('says so, where the caller says nothing else', () => {
    expect(mountEmptyPane().text()).toContain('Nothing open')
  })

  it('is filled by what the caller draws', () => {
    const target = mountEmptyPane({ silence: '<p class="welcome">Welcome</p>' })

    expect(target.find('.welcome').exists()).toBe(true)
    expect(target.text()).not.toContain('Nothing open')
  })

  it('remains over no panel', () => {
    expect(panels(mountEmptyPane())).toHaveLength(0)
  })
})

describe('walking the strip', () => {
  it('steps along with the arrows, and shows what it reaches', async () => {
    const target = mountPane()
    await strip(target)[1]?.trigger('keydown', { key: 'ArrowRight' })

    expect(target.emitted('choose')).toStrictEqual([['notes']])
    expect(getFocusedTab()).toBe('notes')
  })

  it('meets its own ends', async () => {
    const target = mountPane({ pane: pane('main', ['plex', 'chat', 'notes'], 'plex') })
    await strip(target)[0]?.trigger('keydown', { key: 'ArrowLeft' })

    expect(target.emitted('choose')).toStrictEqual([['notes']])
    expect(getFocusedTab()).toBe('notes')
  })

  it('goes to either end by Home and End', async () => {
    const target = mountPane()
    await strip(target)[1]?.trigger('keydown', { key: 'Home' })
    await strip(target)[0]?.trigger('keydown', { key: 'End' })

    expect(target.emitted('choose')).toStrictEqual([['plex'], ['notes']])
  })

  it('leaves every other key to whatever else wants it', async () => {
    const target = mountPane()
    await strip(target)[1]?.trigger('keydown', { key: 'ArrowDown' })
    await strip(target)[1]?.trigger('keydown', { key: 'a' })

    expect(target.emitted('choose')).toBeUndefined()
  })
})

describe('the way out of a panel', () => {
  const inside = '<button class="inside">inside</button>'

  it('is Escape, which lands on the tab the panel is retained under', async () => {
    const tab = mountPane({}, { tab: inside })
    const first = tab.findAll<HTMLButtonElement>('.inside')[1]

    first?.element.focus()
    await first?.trigger('keydown', { key: 'Escape' })

    expect(getFocusedTab()).toBe('chat')
  })

  // A tab closed ahead of the one showing hands it a place in the strip it did
  // not have, and the way out lands on the tab and not on the place.
  it('lands on the tab the panel is retained under after a tab before it closed', async () => {
    const tab = mountPane({}, { tab: inside })
    await tab.setProps({ pane: pane('main', ['chat', 'notes'], 'chat') })

    const first = tab.findAll<HTMLButtonElement>('.inside')[0]
    first?.element.focus()
    await first?.trigger('keydown', { key: 'Escape' })

    expect(getFocusedTab()).toBe('chat')
  })

  it('leaves Escape alone where the panel has already acted on it', async () => {
    const tab = mountPane({}, { tab: inside })
    const first = tab.findAll<HTMLButtonElement>('.inside')[1]

    first?.element.addEventListener('keydown', (event) => event.preventDefault())
    first?.element.focus()
    await first?.trigger('keydown', { key: 'Escape' })

    expect(document.activeElement).toBe(first?.element)
  })
})

describe('the tab that is showing', () => {
  it('is returned as the pane is drawn', () => {
    expect(mountPane().emitted('show')).toStrictEqual([['chat']])
  })

  it('is returned again when another tab takes its place', async () => {
    const target = mountPane()
    await target.setProps({ pane: pane('main', ['plex', 'chat', 'notes'], 'notes') })

    expect(target.emitted('show')).toStrictEqual([['chat'], ['notes']])
  })

  it('is returned once the panel it remains over is on screen', async () => {
    const onScreen: boolean[] = []
    const onShow = (tab: string) => {
      const index = ['plex', 'chat', 'notes'].indexOf(tab)
      const panel = document.getElementById(`later-panel-${index}`)
      onScreen.push(panel?.hasAttribute('data-showing') ?? false)
    }

    const target = mountPane({ pane: pane('later', ['plex', 'chat', 'notes'], 'chat'), onShow })
    await target.setProps({ pane: pane('later', ['plex', 'chat', 'notes'], 'notes') })

    expect(onScreen).toStrictEqual([true, true])
  })

  it('is nothing at all where the pane holds nothing', () => {
    expect(mountPane({ pane: pane('main', []) }).emitted('show')).toBeUndefined()
  })
})
