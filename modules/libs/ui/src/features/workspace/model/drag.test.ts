/** A tab lifted off its strip, and where letting go of it lands it. */
import { effectScope, shallowRef } from 'vue'
import { afterEach, describe, expect, it } from 'vitest'
import { useTabDrag, type TabDragState } from './drag'
import { getPanes } from '../lib/tree'
import type { Tab, Workspace } from '../lib/node'
import type { Clock } from '@/shared/lib/clock'
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

/**
 * A workspace drawn in a frame at (100, 50), 1000 by 600. `main` is the left half and holds one
 * tab; `aside` is the right half and holds two. A strip is the top 30 of a pane.
 */
function createDrag(
  options: {
    getTab?: typeof getTab
    hasFrame?: boolean
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
  const clock: Clock = { now: () => 0, schedule: () => 0, cancel: () => {} }
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
      onTearOff: options.onTearOff,
    }),
  )!

  /** A tab pressed at one place and dragged to another, then held there. */
  const dragTo = (tab: string, from: [number, number], to: [number, number]): void => {
    drag.press(tab, pointer('pointerdown', ...from))
    window.dispatchEvent(pointer('pointermove', ...to))
  }
  const release = (x: number, y: number): void => {
    window.dispatchEvent(pointer('pointerup', x, y))
  }

  return { drag, workspace, scope, frame, state, main, dragTo, release }
}

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

describe('a press on a tab', () => {
  it('is no drag until the pointer has travelled far enough', () => {
    const { drag, dragTo } = start()
    dragTo('plex', [350, 300], [352, 300])

    expect(drag.hasMoved.value).toBe(false)
    expect(drag.label.value).toBeNull()
    expect(drag.overlay.value).toBeNull()
    expect(drag.dropTarget.value).toBeNull()
  })

  it('is a drag once the pointer has travelled far enough', () => {
    const { drag, dragTo } = start()
    dragTo('plex', [350, 300], [360, 300])

    expect(drag.hasMoved.value).toBe(true)
    expect(drag.position.value).toStrictEqual({ x: 360, y: 300 })
  })

  it('is no drag when made with another button', () => {
    const { drag } = start()
    drag.press('plex', pointer('pointerdown', 350, 300, 2))
    window.dispatchEvent(pointer('pointermove', 400, 300))

    expect(drag.hasMoved.value).toBe(false)
  })
})

describe('a tab being dragged', () => {
  it('is named at the pointer by its title', () => {
    const { drag, dragTo } = start()
    dragTo('plex', [350, 300], [360, 300])

    expect(drag.label.value).toBe('Plex')
  })

  it('is named by its identity where nothing says what it is', () => {
    const { drag, dragTo } = start({ getTab: () => undefined })
    dragTo('plex', [350, 300], [360, 300])

    expect(drag.label.value).toBe('plex')
  })

  it('shows the area a drop would take, in the coordinates of the frame', () => {
    const { drag, dragTo } = start()
    dragTo('plex', [350, 300], [850, 300])

    expect(drag.dropTarget.value).toMatchObject({ kind: 'pane', pane: 'aside', side: 'center' })
    expect(drag.overlay.value).toStrictEqual({ x: 500, y: 0, width: 500, height: 600 })
  })

  it('lands on the side of a pane the pointer is nearest', () => {
    const { drag, dragTo } = start()
    dragTo('plex', [350, 300], [620, 300])

    expect(drag.dropTarget.value).toMatchObject({ kind: 'pane', pane: 'aside', side: 'left' })
    expect(drag.overlay.value).toStrictEqual({ x: 500, y: 0, width: 250, height: 600 })
  })

  it('lands on the outer edge where the pointer is within reach of it', () => {
    const { drag, dragTo } = start()
    dragTo('chat', [650, 60], [105, 300])

    expect(drag.dropTarget.value).toMatchObject({ kind: 'edge', side: 'left' })
    expect(drag.overlay.value).toStrictEqual({ x: 0, y: 0, width: 500, height: 600 })
  })

  it('lands among the tabs of a strip, at the gap the pointer is nearest', () => {
    const { drag, dragTo } = start()
    dragTo('plex', [150, 60], [790, 60])

    expect(drag.dropTarget.value).toMatchObject({ kind: 'strip', pane: 'aside', slot: 2 })
    expect(drag.overlay.value).toStrictEqual({ x: 700, y: 0, width: 0, height: 30 })
  })

  it('lands anywhere in the frame, its edges included', () => {
    const { drag, dragTo } = start()

    for (const [x, y] of [
      [100, 300],
      [1100, 300],
      [350, 650],
    ] as const) {
      dragTo('plex', [350, 300], [x, y])
      expect(drag.dropTarget.value?.kind).toBe('edge')
    }
    dragTo('plex', [350, 300], [150, 50])
    expect(drag.dropTarget.value?.kind).toBe('strip')
  })

  it('lands nowhere outside the frame', () => {
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

  it('lands nowhere where no frame is drawn', () => {
    const { drag, dragTo } = start({ hasFrame: false })
    dragTo('plex', [350, 300], [850, 300])

    expect(drag.dropTarget.value).toBeNull()
  })

  it('lands nowhere over what is no pane', () => {
    const { drag, dragTo, state, frame } = start()
    state.targetElement = () => frame
    dragTo('plex', [350, 300], [850, 300])

    expect(drag.dropTarget.value).toBeNull()
  })

  it('lands nowhere over a pane that has no identity', () => {
    const { drag, dragTo, state, main } = start()
    main.setAttribute('data-workspace-pane', '')
    state.targetElement = () => main
    dragTo('plex', [350, 300], [360, 300])

    expect(drag.dropTarget.value).toBeNull()
  })

  it('lands nowhere where the document has nothing under the pointer', () => {
    const { drag, dragTo, state } = start()
    state.targetElement = () => null
    dragTo('plex', [350, 300], [850, 300])

    expect(drag.hasMoved.value).toBe(true)
    expect(drag.dropTarget.value).toBeNull()
  })
})

describe('letting go of a tab', () => {
  it('puts it into the pane it is over', () => {
    const { workspace, dragTo, release } = start()
    dragTo('plex', [350, 300], [850, 300])
    release(850, 300)

    expect(getTabs(workspace.value)).toStrictEqual([['chat', 'notes', 'plex']])
  })

  it('divides the pane it is over, where it is over a side', () => {
    const { workspace, dragTo, release } = start()
    dragTo('chat', [650, 60], [620, 300])
    release(620, 300)

    expect(getTabs(workspace.value)).toStrictEqual([['plex'], ['chat'], ['notes']])
  })

  it('divides the whole workspace, where it is over an outer edge', () => {
    const { workspace, dragTo, release } = start()
    dragTo('chat', [650, 60], [105, 300])
    release(105, 300)

    expect(getTabs(workspace.value)).toStrictEqual([['chat'], ['plex'], ['notes']])
  })

  it('puts it in order among the tabs of its own strip', () => {
    const { workspace, dragTo, release } = start()
    dragTo('chat', [650, 60], [790, 60])
    release(790, 60)

    expect(getTabs(workspace.value)).toStrictEqual([['plex'], ['notes', 'chat']])
  })

  it('puts it in order among the tabs of another strip, which it joins', () => {
    const { workspace, dragTo, release } = start()
    dragTo('plex', [150, 60], [610, 60])
    release(610, 60)

    expect(getTabs(workspace.value)).toStrictEqual([['plex', 'chat', 'notes']])
  })

  it('counts the pane it was pressed in from the pointer, not from the focus', () => {
    const { workspace, dragTo, release } = start()
    // `chat` is pressed in `aside`, where the focus is `main`.
    dragTo('chat', [650, 60], [790, 60])
    release(790, 60)

    expect(getTabs(workspace.value)).toStrictEqual([['plex'], ['notes', 'chat']])
  })

  it('counts the pane it was pressed in as the focus where the press is over no pane', () => {
    const { workspace, state, drag, release } = start()
    const origTargetElement = state.targetElement
    state.targetElement = () => null
    drag.press('plex', pointer('pointerdown', 150, 60))
    state.targetElement = origTargetElement
    window.dispatchEvent(pointer('pointermove', 610, 60))
    release(610, 60)

    expect(getTabs(workspace.value)).toStrictEqual([['plex', 'chat', 'notes']])
  })

  it('counts the pane it was pressed in from the pointer where the focus is elsewhere', () => {
    const { workspace, dragTo, release } = start()
    workspace.value = { ...workspace.value, focus: 'aside' }
    dragTo('plex', [150, 60], [610, 60])
    release(610, 60)

    expect(getTabs(workspace.value)).toStrictEqual([['plex', 'chat', 'notes']])
  })

  it('leaves the shown tab of a pane as it was when a tab is put in order within it', () => {
    const { workspace, dragTo, release } = start()
    dragTo('notes', [750, 60], [610, 60])
    release(610, 60)

    const aside = getPanes(workspace.value.root).find((each) => each.id === 'aside')
    expect(aside?.tabs).toStrictEqual(['notes', 'chat'])
    expect(aside?.active).toBe('chat')
  })

  it('changes nothing where the pointer is let go over no drop target', () => {
    const { workspace, dragTo, release } = start()
    const before = workspace.value
    dragTo('plex', [350, 300], [20, 300])
    release(20, 300)

    expect(workspace.value).toBe(before)
  })

  it('changes nothing where the press never became a drag', () => {
    const { workspace, drag, release } = start()
    const before = workspace.value
    drag.press('plex', pointer('pointerdown', 350, 300))
    release(350, 300)

    expect(workspace.value).toBe(before)
  })

  it('reports a tab let go over no drop target to onTearOff with screen offset', () => {
    let divided: { tab: string; at: { x: number; y: number } } | null = null
    const originalScreenX = window.screenX
    const originalScreenY = window.screenY
    Object.defineProperty(window, 'screenX', { value: 100, configurable: true })
    Object.defineProperty(window, 'screenY', { value: 50, configurable: true })

    const { dragTo, release } = start({
      onTearOff: (tab, at) => {
        divided = { tab, at }
      },
    })
    dragTo('plex', [350, 300], [20, 300])
    release(20, 300)

    expect(divided).toStrictEqual({ tab: 'plex', at: { x: 120, y: 350 } })

    Object.defineProperty(window, 'screenX', { value: originalScreenX, configurable: true })
    Object.defineProperty(window, 'screenY', { value: originalScreenY, configurable: true })
  })
})
