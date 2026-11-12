import { mount } from '@vue/test-utils'
import { computed } from 'vue'
import { SplitterGroup, SplitterPanel, SplitterResizeHandle } from 'reka-ui'
import { afterEach, describe, expect, it, vi } from 'vitest'
import WorkspaceBranch from './WorkspaceBranch.vue'
import { WorkspacePane } from '../pane'
import { NO_STRIP_EDGE, WORKSPACE_CONTEXT, type WorkspaceContext } from '../../model/context'
import { split, stack } from '../../fixtures/build'
import { type Branch, type Tab, type TabId } from '../../lib/node'

const TITLES: Readonly<Record<string, string>> = { one: 'One', two: 'Two', three: 'Three' }

const twoPanes = (): Branch =>
  split('root', [stack('left', 'one'), stack('right', 'two')], [0.7, 0.3]) as Branch

const createNestedBranch = (): Branch =>
  split('root', [
    stack('left', 'one'),
    split('down', [stack('upper', 'two'), stack('lower', 'three')]),
  ]) as Branch

const mountBranch = (node: Branch, slots: Record<string, string> = {}) => {
  const resize = vi.fn<(branch: string, sizes: readonly number[]) => void>()
  const claim = vi.fn<(pane: string) => void>()

  const workspacing: WorkspaceContext = {
    findTab: (id: TabId): Tab | undefined =>
      TITLES[id] === undefined ? undefined : { id, title: TITLES[id] },
    focus: 'left',
    minimum: 220,
    getStripEdge: () => NO_STRIP_EDGE,
    choose: vi.fn(),
    close: vi.fn(),
    startTabPress: vi.fn(),
    menu: vi.fn(),
    claim,
    resize,
    show: vi.fn(),
  }

  const wrapper = mount(WorkspaceBranch, {
    attachTo: document.body,
    global: { provide: { [WORKSPACE_CONTEXT as symbol]: computed(() => workspacing) } },
    props: { node, axis: 'horizontal' as const, depth: 0 },
    slots: { tab: '<span class="tab-slot">tab</span>', ...slots },
  })

  return { wrapper, resize, claim }
}

const stubResizeObserver = (): ((length: number) => void) => {
  const callbacks: ResizeObserverCallback[] = []

  vi.stubGlobal(
    'ResizeObserver',
    class {
      constructor(tell: ResizeObserverCallback) {
        callbacks.push(tell)
      }
      observe = vi.fn()
      disconnect = vi.fn()
    },
  )

  return (length) => {
    const entries = [{ contentRect: { width: length, height: length } } as ResizeObserverEntry]
    for (const callback of callbacks) callback(entries, {} as ResizeObserver)
  }
}

type BranchFixture = ReturnType<typeof mountBranch>

const panes = ({ wrapper }: BranchFixture) => wrapper.findAllComponents(WorkspacePane)

const groups = ({ wrapper }: BranchFixture) => wrapper.findAllComponents(SplitterGroup)

const getHandles = ({ wrapper }: BranchFixture) => wrapper.findAllComponents(SplitterResizeHandle)

const layout = (one: BranchFixture, pos: number, sizes: number[]) =>
  groups(one)[pos]!.vm.$emit('layout', sizes)

describe('WorkspaceBranch layout', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders a panel for each child and a resize handle between them', () => {
    const one = mountBranch(twoPanes())

    expect(panes(one)).toHaveLength(2)
    expect(getHandles(one)).toHaveLength(1)
  })

  it('renders nested branches and all descendant panes', () => {
    const one = mountBranch(createNestedBranch())

    expect(one.wrapper.findAllComponents(WorkspaceBranch)).toHaveLength(1)
    expect(panes(one)).toHaveLength(3)
    expect(
      one.wrapper
        .findAll('[data-workspace-pane]')
        .map((each) => each.attributes('data-workspace-pane')),
    ).toStrictEqual(['left', 'upper', 'lower'])
  })

  it('alternates splitter direction at each nesting depth', () => {
    const one = mountBranch(createNestedBranch())

    expect(groups(one)[0]?.props('direction')).toBe('horizontal')
    expect(groups(one)[1]?.props('direction')).toBe('vertical')
  })

  it('passes defaultSize percentages and node ids to each panel', () => {
    const one = mountBranch(twoPanes())
    const panels = one.wrapper.findAllComponents(SplitterPanel)

    expect(panels.map((each) => each.props('defaultSize'))).toStrictEqual([70, 30])
    expect(panels.map((each) => each.props('id'))).toStrictEqual(['left', 'right'])
  })

  it('updates panel minSize percentages from branch size and minimum pixel width', async () => {
    const stretch = stubResizeObserver()
    const one = mountBranch(twoPanes())
    const floors = () =>
      one.wrapper.findAllComponents(SplitterPanel).map((each) => each.props('minSize'))

    await one.wrapper.vm.$nextTick()
    stretch(880)
    await one.wrapper.vm.$nextTick()
    expect(floors()).toStrictEqual([25, 25])

    stretch(440)
    await one.wrapper.vm.$nextTick()
    expect(floors()).toStrictEqual([50, 50])

    stretch(200)
    await one.wrapper.vm.$nextTick()
    expect(floors()).toStrictEqual([50, 50])
  })

  it('forwards slots to nested panes', () => {
    const one = mountBranch(createNestedBranch())

    expect(one.wrapper.findAll('.tab-slot')).toHaveLength(3)
  })

  it('marks only the focused pane with data-focused', () => {
    const one = mountBranch(twoPanes())
    const marked = one.wrapper
      .findAll('[data-workspace-pane]')
      .filter((each) => each.attributes('data-focused') !== undefined)

    expect(marked.map((each) => each.attributes('data-workspace-pane'))).toStrictEqual(['left'])
  })
})

describe('resize handle dragging', () => {
  const grab = (one: BranchFixture, now: boolean) => getHandles(one)[0]!.vm.$emit('dragging', now)

  it('does not call resize while the handle is being dragged', async () => {
    const one = mountBranch(twoPanes())

    grab(one, true)
    layout(one, 0, [40, 60])
    await one.wrapper.vm.$nextTick()

    expect(one.resize).not.toHaveBeenCalled()
  })

  it('calls resize once with the final ratios when dragging ends', async () => {
    const one = mountBranch(twoPanes())

    grab(one, true)
    layout(one, 0, [40, 60])
    layout(one, 0, [45, 55])
    grab(one, false)
    await one.wrapper.vm.$nextTick()

    expect(one.resize).toHaveBeenCalledExactlyOnceWith('root', [0.45, 0.55])
  })

  it('calls resize immediately when layout changes without an active drag', async () => {
    const one = mountBranch(twoPanes())

    layout(one, 0, [20, 80])
    await one.wrapper.vm.$nextTick()

    expect(one.resize).toHaveBeenCalledExactlyOnceWith('root', [0.2, 0.8])
  })

  it('does not call resize when layout sizes match the model', async () => {
    const one = mountBranch(twoPanes())

    layout(one, 0, [70, 30])
    await one.wrapper.vm.$nextTick()

    expect(one.resize).not.toHaveBeenCalled()
  })

  it('sets data-resizing on the branch while dragging', async () => {
    const one = mountBranch(twoPanes())
    const branch = one.wrapper.find('.branch')

    grab(one, true)
    await one.wrapper.vm.$nextTick()
    expect(branch.attributes('data-resizing')).toBe('')

    grab(one, false)
    await one.wrapper.vm.$nextTick()
    expect(branch.attributes('data-resizing')).toBeUndefined()
  })

  it('passes the resized branch id for nested branches', async () => {
    const one = mountBranch(createNestedBranch())

    layout(one, 1, [30, 70])
    await one.wrapper.vm.$nextTick()

    expect(one.resize).toHaveBeenCalledExactlyOnceWith('down', [0.3, 0.7])
  })

  it('does not call resize when unmounted during a drag', () => {
    const one = mountBranch(twoPanes())

    grab(one, true)
    layout(one, 0, [40, 60])
    one.wrapper.unmount()

    expect(one.resize).not.toHaveBeenCalled()
  })
})
