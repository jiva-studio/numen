/**
 * The splitter that composes panes, and what a handle reports back.
 *
 * The shares a handle settles on are what the workspace is told; the
 * arithmetic on them is `shares.test.ts`.
 */
import { mount } from '@vue/test-utils'
import { computed } from 'vue'
import { SplitterGroup, SplitterPanel, SplitterResizeHandle } from 'reka-ui'
import { afterEach, describe, expect, it, vi } from 'vitest'
import WorkspaceBranch from './WorkspaceBranch.vue'
import { WorkspacePane } from '../pane'
import { WORKSPACE_CONTEXT, type WorkspaceContext } from '../../model/context'
import { split, stack } from '../../fixtures/build'
import { type Branch, type Tab, type TabId } from '../../lib/node'

const TITLES: Readonly<Record<string, string>> = { one: 'One', two: 'Two', three: 'Three' }

/** Two panes side by side, in unequal shares. */
const twoPanes = (): Branch =>
  split('root', [stack('left', 'one'), stack('right', 'two')], [0.7, 0.3]) as Branch

/** A pane beside a branch of two, so a branch holds a branch. */
const createNested = (): Branch =>
  split('root', [
    stack('left', 'one'),
    split('down', [stack('upper', 'two'), stack('lower', 'three')]),
  ]) as Branch

const mountBranch = (node: Branch, slots: Record<string, string> = {}) => {
  const resize = vi.fn<(branch: string, sizes: readonly number[]) => void>()
  const claim = vi.fn<(pane: string) => void>()

  const workspacing: WorkspaceContext = {
    tabOf: (id: TabId): Tab | undefined =>
      TITLES[id] === undefined ? undefined : { id, title: TITLES[id] },
    focus: 'left',
    minimum: 220,
    choose: () => {},
    close: () => {},
    lift: () => {},
    claim,
    resize,
    show: () => {},
  }

  const held = mount(WorkspaceBranch, {
    attachTo: document.body,
    global: { provide: { [WORKSPACE_CONTEXT as symbol]: computed(() => workspacing) } },
    props: { node, axis: 'horizontal' as const, depth: 0 },
    slots: { tab: '<span class="held">held</span>', ...slots },
  })

  return { held, resize, claim }
}

/**
 * How long the branch is on screen, which is what the smallest share is worked
 * out from. Nothing here has a size, so the observer the branch follows its own
 * element with is stood in for. Call it before mounting.
 */
const stubResizeObserver = (): ((length: number) => void) => {
  const told: ResizeObserverCallback[] = []

  vi.stubGlobal(
    'ResizeObserver',
    class {
      constructor(tell: ResizeObserverCallback) {
        told.push(tell)
      }
      observe(): void {}
      disconnect(): void {}
    },
  )

  return (length) => {
    const seen = [{ contentRect: { width: length, height: length } } as ResizeObserverEntry]
    for (const tell of told) tell(seen, {} as ResizeObserver)
  }
}

type BranchFixture = ReturnType<typeof mountBranch>

const panes = ({ held }: BranchFixture) => held.findAllComponents(WorkspacePane)

const groups = ({ held }: BranchFixture) => held.findAllComponents(SplitterGroup)

const getHandles = ({ held }: BranchFixture) => held.findAllComponents(SplitterResizeHandle)

/** The splitter reporting the shares it has settled on, in percent. */
const layout = (one: BranchFixture, at: number, sizes: number[]) =>
  groups(one)[at]!.vm.$emit('layout', sizes)

describe('what a branch draws', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('is a panel for each child, with a handle between them', () => {
    const one = mountBranch(twoPanes())

    expect(panes(one)).toHaveLength(2)
    expect(getHandles(one)).toHaveLength(1)
  })

  it('nests a branch inside a branch, and draws every pane of it', () => {
    const one = mountBranch(createNested())

    expect(one.held.findAllComponents(WorkspaceBranch)).toHaveLength(1)
    expect(panes(one)).toHaveLength(3)
    expect(
      one.held
        .findAll('[data-workspace-pane]')
        .map((each) => each.attributes('data-workspace-pane')),
    ).toStrictEqual(['left', 'upper', 'lower'])
  })

  it('turns a quarter at each level down', () => {
    const one = mountBranch(createNested())

    expect(groups(one)[0]?.props('direction')).toBe('horizontal')
    expect(groups(one)[1]?.props('direction')).toBe('vertical')
  })

  it('gives each panel the share the model holds', () => {
    const one = mountBranch(twoPanes())
    const panels = one.held.findAllComponents(SplitterPanel)

    expect(panels.map((each) => each.props('defaultSize'))).toStrictEqual([70, 30])
    expect(panels.map((each) => each.props('id'))).toStrictEqual(['left', 'right'])
  })

  it('leaves each panel a share of the branch worth drawing in', async () => {
    const stretch = stubResizeObserver()
    const one = mountBranch(twoPanes())
    const floors = () =>
      one.held.findAllComponents(SplitterPanel).map((each) => each.props('minSize'))

    // This workspace calls 220 pixels worth drawing in, which is a quarter of a
    // branch 880 across and half of one 440 across.
    await one.held.vm.$nextTick()
    stretch(880)
    await one.held.vm.$nextTick()
    expect(floors()).toStrictEqual([25, 25])

    stretch(440)
    await one.held.vm.$nextTick()
    expect(floors()).toStrictEqual([50, 50])

    // An equal share is the ceiling: a branch too short to give both children
    // that many pixels divides what it has evenly.
    stretch(200)
    await one.held.vm.$nextTick()
    expect(floors()).toStrictEqual([50, 50])
  })

  it('hands a slot down however deep the pane stands', () => {
    const one = mountBranch(createNested())

    expect(one.held.findAll('.held')).toHaveLength(3)
  })

  it('marks the pane the workspace is focused on', () => {
    const one = mountBranch(twoPanes())
    const marked = one.held
      .findAll('[data-workspace-pane]')
      .filter((each) => each.attributes('data-focused') !== undefined)

    expect(marked.map((each) => each.attributes('data-workspace-pane'))).toStrictEqual(['left'])
  })
})

describe('a handle taken up and put down', () => {
  const grab = (one: BranchFixture, now: boolean) => getHandles(one)[0]!.vm.$emit('dragging', now)

  it('says nothing while it is still held', async () => {
    const one = mountBranch(twoPanes())

    grab(one, true)
    layout(one, 0, [40, 60])
    await one.held.vm.$nextTick()

    expect(one.resize).not.toHaveBeenCalled()
  })

  it('says where it came to rest, once', async () => {
    const one = mountBranch(twoPanes())

    grab(one, true)
    layout(one, 0, [40, 60])
    layout(one, 0, [45, 55])
    grab(one, false)
    await one.held.vm.$nextTick()

    expect(one.resize).toHaveBeenCalledExactlyOnceWith('root', [0.45, 0.55])
  })

  it('says a change the splitter made on its own at once', async () => {
    const one = mountBranch(twoPanes())

    layout(one, 0, [20, 80])
    await one.held.vm.$nextTick()

    expect(one.resize).toHaveBeenCalledExactlyOnceWith('root', [0.2, 0.8])
  })

  it('says nothing for shares the model already holds', async () => {
    const one = mountBranch(twoPanes())

    layout(one, 0, [70, 30])
    await one.held.vm.$nextTick()

    expect(one.resize).not.toHaveBeenCalled()
  })

  it('marks the branch for as long as the handle is held', async () => {
    const one = mountBranch(twoPanes())
    const branch = one.held.find('.branch')

    grab(one, true)
    await one.held.vm.$nextTick()
    expect(branch.attributes('data-resizing')).toBe('')

    grab(one, false)
    await one.held.vm.$nextTick()
    expect(branch.attributes('data-resizing')).toBeUndefined()
  })

  it('names the branch it belongs to and no other', async () => {
    const one = mountBranch(createNested())

    layout(one, 1, [30, 70])
    await one.held.vm.$nextTick()

    expect(one.resize).toHaveBeenCalledExactlyOnceWith('down', [0.3, 0.7])
  })

  it('says nothing about a handle let go of after the branch has gone', () => {
    const one = mountBranch(twoPanes())

    grab(one, true)
    layout(one, 0, [40, 60])
    one.held.unmount()

    expect(one.resize).not.toHaveBeenCalled()
  })
})
