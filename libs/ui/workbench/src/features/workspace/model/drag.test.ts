import { effectScope, shallowRef } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useTabDrag, type TabDragState } from './drag'
import { getPanes } from '../lib/tree'
import type { Tab, Workspace } from '../lib/node'
import type { Clock } from '@numen/ui'
import { createIdFactory, withSpare } from '../fixtures/build'

const TITLES: Readonly<Record<string, string>> = { plex: 'Plex', chat: 'Chat', notes: 'Notes' }

const pointer = (type: string, x: number, y: number, button = 0) =>
  new PointerEvent(type, { clientX: x, clientY: y, button, bubbles: true })

const createBox = (
  tag: string,
  attribute: [string, string] | null,
  box: [number, number, number, number],
): HTMLElement => {
  const element = document.createElement(tag)
  if (attribute) element.setAttribute(attribute[0], attribute[1])
  element.getBoundingClientRect = () => new DOMRect(...box)
  return element
}

const getTab = (id: string): Tab | undefined => {
  const title = TITLES[id]
  return title === undefined ? undefined : { id, title }
}

function createDrag(
  options: {
    getTab?: typeof getTab
    hasFrame?: boolean
    viewport?: { width: number; height: number }
    onTearOff?: (tab: string, pos: { x: number; y: number }) => void
  } = {},
) {
  const frame = createBox('div', null, [100, 50, 1000, 600])
  const main = createBox('div', ['data-workspace-pane', 'main'], [100, 50, 500, 600])
  const aside = createBox('div', ['data-workspace-pane', 'aside'], [600, 50, 500, 600])
  const mainStrip = createBox('div', ['data-workspace-strip', ''], [100, 50, 500, 30])
  const asideStrip = createBox('div', ['data-workspace-strip', ''], [600, 50, 500, 30])
  const plex = createBox('button', null, [100, 50, 100, 30])
  const chat = createBox('button', null, [600, 50, 100, 30])
  const notes = createBox('button', null, [700, 50, 100, 30])
  mainStrip.append(plex)
  asideStrip.append(chat, notes)
  main.append(mainStrip)
  aside.append(asideStrip)
  frame.append(main, aside)
  document.body.append(frame)

  const state = {
    targetElement: (x: number, y: number): Element | null => {
      const tabs = [plex, chat, notes]
      const targetTab = tabs.find((tab) => {
        const box = tab.getBoundingClientRect()
        return x >= box.left && x < box.right && y >= box.top && y < box.bottom
      })
      return targetTab ?? (x < 600 ? main : aside)
    },
  }
  document.elementFromPoint = (x: number, y: number) => state.targetElement(x, y)

  const workspace = shallowRef<Workspace>(withSpare())
  const clock: Clock = { now: () => 0, schedule: () => 0, cancel: vi.fn() }
  const scope = effectScope()
  const drag = scope.run((): TabDragState =>
    useTabDrag({
      workspace,
      frame: shallowRef(options.hasFrame === false ? null : frame),
      getTab: options.getTab ?? getTab,
      getIdFactory: () => createIdFactory(),
      getEdge: () => 10,
      getThreshold: () => 4,
      getClock: () => clock,
      getViewport: () => options.viewport ?? { width: 1100, height: 650 },
      onTearOff: options.onTearOff,
    }),
  )!

  const dragTo = (tab: string, from: [number, number], to: [number, number]): void => {
    drag.press(tab, pointer('pointerdown', ...from))
    window.dispatchEvent(pointer('pointermove', ...to))
  }
  const release = (x: number, y: number): void => {
    window.dispatchEvent(pointer('pointerup', x, y))
  }
  const cancel = (x: number, y: number): void => {
    window.dispatchEvent(pointer('pointercancel', x, y))
  }

  return { drag, workspace, scope, frame, state, main, dragTo, release, cancel }
}

const OUTSIDE_SIDES: readonly [string, [number, number], [number, number]][] = [
  ['left', [105, 300], [-1, 300]],
  ['right', [1095, 300], [1100, 300]],
  ['top', [350, 55], [350, -1]],
  ['bottom', [350, 645], [350, 650]],
]

const getTabs = (workspace: Workspace) => getPanes(workspace.root).map((each) => each.tabs)

let target: ReturnType<typeof createDrag> | null = null

const start = (options?: Parameters<typeof createDrag>[0]) => {
  target = createDrag(options)
  return target
}

afterEach(() => {
  target?.scope.stop()
  target?.frame.remove()
  target = null
  Reflect.deleteProperty(document, 'elementFromPoint')
})

describe('tab press threshold', () => {
  it('does not start a drag before the pointer moves past the threshold', () => {
    const { drag, dragTo } = start()
    dragTo('plex', [350, 300], [352, 300])

    expect(drag.hasMoved.value).toBe(false)
    expect(drag.label.value).toBeNull()
    expect(drag.overlay.value).toBeNull()
    expect(drag.dropTarget.value).toBeNull()
  })

  it('starts a drag after the pointer moves past the threshold', () => {
    const { drag, dragTo } = start()
    dragTo('plex', [350, 300], [360, 300])

    expect(drag.hasMoved.value).toBe(true)
    expect(drag.position.value).toStrictEqual({ x: 360, y: 300 })
  })

  it('ignores pointerdown from non-primary buttons', () => {
    const { drag } = start()
    drag.press('plex', pointer('pointerdown', 350, 300, 2))
    window.dispatchEvent(pointer('pointermove', 400, 300))

    expect(drag.hasMoved.value).toBe(false)
  })
})

describe('active tab drag', () => {
  it('uses the tab title for the drag label', () => {
    const { drag, dragTo } = start()
    dragTo('plex', [350, 300], [360, 300])

    expect(drag.label.value).toBe('Plex')
  })

  it('falls back to the tab id when getTab returns undefined', () => {
    const { drag, dragTo } = start({ getTab: () => undefined })
    dragTo('plex', [350, 300], [360, 300])

    expect(drag.label.value).toBe('plex')
  })

  it('computes the drop overlay in frame coordinates', () => {
    const { drag, dragTo } = start()
    dragTo('plex', [350, 300], [850, 300])

    expect(drag.dropTarget.value).toMatchObject({ kind: 'pane', pane: 'aside', side: 'center' })
    expect(drag.overlay.value).toStrictEqual({ x: 500, y: 0, width: 500, height: 600 })
  })

  it('targets the nearest pane side when over a pane border zone', () => {
    const { drag, dragTo } = start()
    dragTo('plex', [350, 300], [620, 300])

    expect(drag.dropTarget.value).toMatchObject({ kind: 'pane', pane: 'aside', side: 'left' })
    expect(drag.overlay.value).toStrictEqual({ x: 500, y: 0, width: 250, height: 600 })
  })

  it('targets the workspace outer edge when inside the edge band', () => {
    const { drag, dragTo } = start()
    dragTo('chat', [650, 60], [105, 300])

    expect(drag.dropTarget.value).toMatchObject({ kind: 'edge', side: 'left' })
    expect(drag.overlay.value).toStrictEqual({ x: 0, y: 0, width: 500, height: 600 })
  })

  it('targets the nearest tab slot when over a tab strip', () => {
    const { drag, dragTo } = start()
    dragTo('plex', [150, 60], [790, 60])

    expect(drag.dropTarget.value).toMatchObject({ kind: 'strip', pane: 'aside', slot: 2 })
    expect(drag.overlay.value).toStrictEqual({ x: 700, y: 0, width: 0, height: 30 })
  })

  it('returns a drop target at the frame bounds inside the viewport', () => {
    const { drag, dragTo } = start()

    for (const [x, y] of [
      [100, 300],
      [1099, 300],
      [350, 649],
    ] as const) {
      dragTo('plex', [350, 300], [x, y])
      expect(drag.dropTarget.value?.kind).toBe('edge')
    }
    dragTo('plex', [350, 300], [150, 50])
    expect(drag.dropTarget.value?.kind).toBe('strip')
  })

  it('returns a null drop target outside the frame bounds', () => {
    const { drag, dragTo } = start()

    for (const [x, y] of [
      [99, 300],
      [1101, 300],
      [350, 49],
      [350, 651],
    ] as const) {
      dragTo('plex', [350, 300], [x, y])
      expect(drag.hasMoved.value).toBe(true)
      expect(drag.dropTarget.value).toBeNull()
      expect(drag.overlay.value).toBeNull()
    }
  })

  it('returns a null drop target when frame is null', () => {
    const { drag, dragTo } = start({ hasFrame: false })
    dragTo('plex', [350, 300], [850, 300])

    expect(drag.dropTarget.value).toBeNull()
  })

  it('returns a null drop target when the element under the pointer is not in a pane', () => {
    const { drag, dragTo, state, frame } = start()
    state.targetElement = () => frame
    dragTo('plex', [350, 300], [850, 300])

    expect(drag.dropTarget.value).toBeNull()
  })

  it('returns a null drop target when the pane has an empty data-workspace-pane attribute', () => {
    const { drag, dragTo, state, main } = start()
    main.setAttribute('data-workspace-pane', '')
    state.targetElement = () => main
    dragTo('plex', [350, 300], [360, 300])

    expect(drag.dropTarget.value).toBeNull()
  })

  it('returns a null drop target when elementFromPoint returns null', () => {
    const { drag, dragTo, state } = start()
    state.targetElement = () => null
    dragTo('plex', [350, 300], [850, 300])

    expect(drag.hasMoved.value).toBe(true)
    expect(drag.dropTarget.value).toBeNull()
  })
})

describe('dropping a tab', () => {
  it('moves the tab into the target pane when dropped on its center', () => {
    const { workspace, dragTo, release } = start()
    dragTo('plex', [350, 300], [850, 300])
    release(850, 300)

    expect(getTabs(workspace.value)).toStrictEqual([['chat', 'notes', 'plex']])
  })

  it('splits the target pane when dropped on a pane side', () => {
    const { workspace, dragTo, release } = start()
    dragTo('chat', [650, 60], [620, 300])
    release(620, 300)

    expect(getTabs(workspace.value)).toStrictEqual([['plex'], ['chat'], ['notes']])
  })

  it('splits the workspace root when dropped on an outer edge', () => {
    const { workspace, dragTo, release } = start()
    dragTo('chat', [650, 60], [105, 300])
    release(105, 300)

    expect(getTabs(workspace.value)).toStrictEqual([['chat'], ['plex'], ['notes']])
  })

  it('reorders the tab within its own strip', () => {
    const { workspace, dragTo, release } = start()
    dragTo('chat', [650, 60], [790, 60])
    release(790, 60)

    expect(getTabs(workspace.value)).toStrictEqual([['plex'], ['notes', 'chat']])
  })

  it('moves the tab into another strip at the target slot', () => {
    const { workspace, dragTo, release } = start()
    dragTo('plex', [150, 60], [610, 60])
    release(610, 60)

    expect(getTabs(workspace.value)).toStrictEqual([['plex', 'chat', 'notes']])
  })

  it('resolves the source pane from the pointerdown position', () => {
    const { workspace, dragTo, release } = start()
    dragTo('chat', [650, 60], [790, 60])
    release(790, 60)

    expect(getTabs(workspace.value)).toStrictEqual([['plex'], ['notes', 'chat']])
  })

  it('falls back to the focused pane when pointerdown is not over a pane', () => {
    const { workspace, state, drag, release } = start()
    const origTargetElement = state.targetElement
    state.targetElement = () => null
    drag.press('plex', pointer('pointerdown', 150, 60))
    state.targetElement = origTargetElement
    window.dispatchEvent(pointer('pointermove', 610, 60))
    release(610, 60)

    expect(getTabs(workspace.value)).toStrictEqual([['plex', 'chat', 'notes']])
  })

  it('moves a tab from an unfocused pane to another strip', () => {
    const { workspace, dragTo, release } = start()
    workspace.value = { ...workspace.value, focus: 'aside' }
    dragTo('plex', [150, 60], [610, 60])
    release(610, 60)

    expect(getTabs(workspace.value)).toStrictEqual([['plex', 'chat', 'notes']])
  })

  it('preserves the active tab when reordering within the same strip', () => {
    const { workspace, dragTo, release } = start()
    dragTo('notes', [750, 60], [610, 60])
    release(610, 60)

    const aside = getPanes(workspace.value.root).find((each) => each.id === 'aside')
    expect(aside?.tabs).toStrictEqual(['notes', 'chat'])
    expect(aside?.active).toBe('chat')
  })

  it('leaves the workspace unchanged when pointerup is over no drop target', () => {
    const { workspace, dragTo, release } = start()
    const before = workspace.value
    dragTo('plex', [350, 300], [20, 300])
    release(20, 300)

    expect(workspace.value).toBe(before)
  })

  it('leaves the workspace unchanged when pointerup occurs before the drag threshold', () => {
    const { workspace, drag, release } = start()
    const before = workspace.value
    drag.press('plex', pointer('pointerdown', 350, 300))
    release(350, 300)

    expect(workspace.value).toBe(before)
  })

  it('does not tear off the tab on pointerup inside the viewport outside the frame', () => {
    const tearOffs: string[] = []
    const { workspace, dragTo, release } = start({ onTearOff: (tab) => tearOffs.push(tab) })
    const before = workspace.value
    dragTo('plex', [350, 300], [20, 300])
    release(20, 300)

    expect(workspace.value).toBe(before)
    expect(tearOffs).toStrictEqual([])
  })

  it('passes the pointerup position in screen coordinates to onTearOff', () => {
    let divided: { tab: string; at: { x: number; y: number } } | null = null
    const originalScreenX = window.screenX
    const originalScreenY = window.screenY
    Object.defineProperty(window, 'screenX', { value: 100, configurable: true })
    Object.defineProperty(window, 'screenY', { value: 50, configurable: true })

    const { dragTo, release } = start({
      onTearOff: (tab, targetPos) => {
        divided = { tab, at: targetPos }
      },
    })
    dragTo('plex', [350, 300], [20, 300])
    release(-20, 300)

    expect(divided).toStrictEqual({ tab: 'plex', at: { x: 80, y: 350 } })

    Object.defineProperty(window, 'screenX', { value: originalScreenX, configurable: true })
    Object.defineProperty(window, 'screenY', { value: originalScreenY, configurable: true })
  })

  it.each(OUTSIDE_SIDES)(
    'tears off the tab on pointerup past the %s side of the viewport after a move in the edge band',
    (_side, last, end) => {
      const tearOffs: { tab: string; at: { x: number; y: number } }[] = []
      const { workspace, drag, dragTo, release } = start({
        onTearOff: (tab, pos) => tearOffs.push({ tab, at: pos }),
      })
      const before = workspace.value
      dragTo('chat', [650, 60], last)
      expect(drag.dropTarget.value?.kind).toBe('edge')

      release(...end)

      expect(workspace.value).toBe(before)
      expect(tearOffs).toStrictEqual([
        { tab: 'chat', at: { x: window.screenX + end[0], y: window.screenY + end[1] } },
      ])
    },
  )

  it('tears off the tab on pointerup outside the viewport over a pane that extends past it', () => {
    const tearOffs: string[] = []
    const { workspace, dragTo, release } = start({
      viewport: { width: 1000, height: 600 },
      onTearOff: (tab) => tearOffs.push(tab),
    })
    const before = workspace.value
    dragTo('plex', [350, 300], [850, 300])
    release(1050, 300)

    expect(workspace.value).toBe(before)
    expect(tearOffs).toStrictEqual(['plex'])
  })

  it('splits the workspace on pointerup in the edge band after a move outside the viewport', () => {
    const tearOffs: string[] = []
    const { workspace, dragTo, release } = start({ onTearOff: (tab) => tearOffs.push(tab) })
    dragTo('chat', [650, 60], [-1, 300])
    release(105, 300)

    expect(getTabs(workspace.value)).toStrictEqual([['chat'], ['plex'], ['notes']])
    expect(tearOffs).toStrictEqual([])
  })

  it('neither drops nor tears off the tab on pointercancel', () => {
    const tearOffs: string[] = []
    const { workspace, drag, dragTo, cancel } = start({ onTearOff: (tab) => tearOffs.push(tab) })
    const before = workspace.value
    dragTo('chat', [650, 60], [105, 300])
    cancel(-1, 300)

    expect(workspace.value).toBe(before)
    expect(tearOffs).toStrictEqual([])
    expect(drag.dropTarget.value).toBeNull()
  })
})
