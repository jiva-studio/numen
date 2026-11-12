import { mount } from '@vue/test-utils'
import { computed } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import WorkspacePane from './WorkspacePane.vue'
import { NO_STRIP_EDGE, WORKSPACE_CONTEXT, type WorkspaceContext } from '../../model/context'
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
  getStripEdge: () => NO_STRIP_EDGE,
  choose: vi.fn(),
  close: vi.fn(),
  startTabPress: vi.fn(),
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

describe('tab strip', () => {
  it('renders a tablist and links each tab to its tabpanel', () => {
    const target = mountPane()

    expect(target.find('[role="tablist"]').exists()).toBe(true)
    expect(strip(target)).toHaveLength(3)
    expect(panels(target)).toHaveLength(3)

    for (const [at, tab] of strip(target).entries()) {
      expect(tab.attributes('aria-controls')).toBe(panels(target)[at]?.attributes('id'))
      expect(panels(target)[at]?.attributes('aria-labelledby')).toBe(tab.attributes('id'))
    }
  })

  it('sets aria-selected on the active tab', () => {
    const selected = strip(mountPane()).map((tab) => tab.attributes('aria-selected'))
    expect(selected).toStrictEqual(['false', 'true', 'false'])
  })

  it('sets tabindex to 0 only on the active tab', () => {
    const stops = strip(mountPane()).map((tab) => tab.attributes('tabindex'))
    expect(stops).toStrictEqual(['-1', '0', '-1'])
  })

  it('passes each tab mark from the workspace context', () => {
    const target = mountPane({ marks: { chat: 'unsaved' } })
    const marks = strip(target).map((tab) => {
      const renderedMark = tab.find('.tab__mark')
      return renderedMark.exists() ? renderedMark.attributes('aria-label') : null
    })

    expect(marks).toStrictEqual([null, 'unsaved', null])
  })

  it('renders the strip when the pane has tabs', () => {
    expect(mountPane().find('[data-workspace-strip]').exists()).toBe(true)
  })

  it('omits the strip when the pane has no tabs', () => {
    expect(
      mountPane({ pane: pane('main', []) })
        .find('[data-workspace-strip]')
        .exists(),
    ).toBe(false)
  })
})

describe('empty pane', () => {
  const mountEmptyPane = (slots: Record<string, string> = {}) =>
    mountPane({ pane: pane('main', []) }, slots)

  it('renders default empty text when no silence slot is provided', () => {
    expect(mountEmptyPane().text()).toContain('Nothing open')
  })

  it('renders the silence slot when provided', () => {
    const target = mountEmptyPane({ silence: '<p class="welcome">Welcome</p>' })

    expect(target.find('.welcome').exists()).toBe(true)
    expect(target.text()).not.toContain('Nothing open')
  })

  it('renders no tabpanels when the pane is empty', () => {
    expect(panels(mountEmptyPane())).toHaveLength(0)
  })
})

describe('strip keyboard navigation', () => {
  it('moves focus and emits choose on ArrowRight', async () => {
    const target = mountPane()
    await strip(target)[1]?.trigger('keydown', { key: 'ArrowRight' })

    expect(target.emitted('choose')).toStrictEqual([['notes']])
    expect(getFocusedTab()).toBe('notes')
  })

  it('wraps around from the first tab to the last on ArrowLeft', async () => {
    const target = mountPane({ pane: pane('main', ['plex', 'chat', 'notes'], 'plex') })
    await strip(target)[0]?.trigger('keydown', { key: 'ArrowLeft' })

    expect(target.emitted('choose')).toStrictEqual([['notes']])
    expect(getFocusedTab()).toBe('notes')
  })

  it('selects the first and last tab on Home and End', async () => {
    const target = mountPane()
    await strip(target)[1]?.trigger('keydown', { key: 'Home' })
    await strip(target)[0]?.trigger('keydown', { key: 'End' })

    expect(target.emitted('choose')).toStrictEqual([['plex'], ['notes']])
  })

  it('ignores non-navigation keys', async () => {
    const target = mountPane()
    await strip(target)[1]?.trigger('keydown', { key: 'ArrowDown' })
    await strip(target)[1]?.trigger('keydown', { key: 'a' })

    expect(target.emitted('choose')).toBeUndefined()
  })
})

describe('panel Escape key', () => {
  const inside = '<button class="inside">inside</button>'

  it('moves focus from the panel to the active tab on Escape', async () => {
    const tab = mountPane({}, { tab: inside })
    const first = tab.findAll<HTMLButtonElement>('.inside')[1]

    first?.element.focus()
    await first?.trigger('keydown', { key: 'Escape' })

    expect(getFocusedTab()).toBe('chat')
  })

  it('focuses the active tab after a preceding tab is removed', async () => {
    const tab = mountPane({}, { tab: inside })
    await tab.setProps({ pane: pane('main', ['chat', 'notes'], 'chat') })

    const first = tab.findAll<HTMLButtonElement>('.inside')[0]
    first?.element.focus()
    await first?.trigger('keydown', { key: 'Escape' })

    expect(getFocusedTab()).toBe('chat')
  })

  it('ignores Escape when defaultPrevented is true', async () => {
    const tab = mountPane({}, { tab: inside })
    const first = tab.findAll<HTMLButtonElement>('.inside')[1]

    first?.element.addEventListener('keydown', (event) => event.preventDefault())
    first?.element.focus()
    await first?.trigger('keydown', { key: 'Escape' })

    expect(document.activeElement).toBe(first?.element)
  })
})

describe('show event', () => {
  it('emits show with the active tab on mount', () => {
    expect(mountPane().emitted('show')).toStrictEqual([['chat']])
  })

  it('emits show when the active tab changes', async () => {
    const target = mountPane()
    await target.setProps({ pane: pane('main', ['plex', 'chat', 'notes'], 'notes') })

    expect(target.emitted('show')).toStrictEqual([['chat'], ['notes']])
  })

  it('emits show after the active panel is marked data-showing', async () => {
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

  it('does not emit show when the pane has no active tab', () => {
    expect(mountPane({ pane: pane('main', []) }).emitted('show')).toBeUndefined()
  })
})
