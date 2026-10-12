/** A tab lifted off its strip, and where letting go of it lands it. */
import { effectScope, shallowRef } from 'vue'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { useTabDrag, type TabDragState } from './drag'
import { panesOf } from '../lib/tree'
import type { Tab, Workspace } from '../lib/node'
import type { Clock } from '@/shared/lib/clock'
import { createIdFactory, sideBySide } from '../fixtures/build'

const TITLES: Readonly<Record<string, string>> = { plex: 'Plex', chat: 'Chat' }

const pointer = (type: string, x: number, y: number, button = 0) =>
  new PointerEvent(type, { clientX: x, clientY: y, button, bubbles: true })

const createBox = (attribute: string | null, x: number, width: number): HTMLElement => {
  const element = document.createElement('div')
  if (attribute !== null) element.setAttribute('data-workspace-pane', attribute)
  element.getBoundingClientRect = () => new DOMRect(x, 50, width, 600)
  return element
}

/** A workspace drawn in a frame at (100, 50): `main` to the left of x = 600 and `aside` to the right. */
function createDrag(getTab: (id: string) => Tab | undefined) {
  const frame = createBox(null, 100, 1000)
  const main = createBox('main', 100, 500)
  const aside = createBox('aside', 600, 500)
  frame.append(main, aside)
  document.body.append(frame)
  document.elementFromPoint = (x: number): Element => (x < 600 ? main : aside)

  const workspace = shallowRef<Workspace>(sideBySide())
  const clock: Clock = { now: () => 0, schedule: () => 0, cancel: () => {} }
  const scope = effectScope()
  const drag = scope.run(
    (): TabDragState =>
      useTabDrag({
        workspace,
        frame: shallowRef(frame),
        getTab,
        getIdFactory: () => createIdFactory(),
        getEdge: () => 10,
        getThreshold: () => 4,
        getClock: () => clock,
      }),
  )!

  return { drag, workspace, scope, frame }
}

const tabsOf = (workspace: Workspace, pane: string) =>
  panesOf(workspace.root).find((each) => each.id === pane)?.tabs

const getTab = (id: string): Tab | undefined => {
  const title = TITLES[id]
  return title === undefined ? undefined : { id, title }
}

let held: ReturnType<typeof createDrag> | null = null

beforeEach(() => {
  held = null
})

afterEach(() => {
  held?.scope.stop()
  held?.frame.remove()
  Reflect.deleteProperty(document, 'elementFromPoint')
})

describe('a press on a tab', () => {
  it('is no drag until the pointer has travelled far enough', () => {
    held = createDrag(getTab)
    held.drag.press('plex', pointer('pointerdown', 350, 300))
    window.dispatchEvent(pointer('pointermove', 352, 300))

    expect(held.drag.hasMoved.value).toBe(false)
    expect(held.drag.label.value).toBeNull()
    expect(held.drag.overlay.value).toBeNull()
    expect(held.drag.landing.value).toBeNull()
  })

  it('is a drag once the pointer has travelled far enough', () => {
    held = createDrag(getTab)
    held.drag.press('plex', pointer('pointerdown', 350, 300))
    window.dispatchEvent(pointer('pointermove', 360, 300))

    expect(held.drag.hasMoved.value).toBe(true)
    expect(held.drag.position.value).toStrictEqual({ x: 360, y: 300 })
  })

  it('is no drag when made with another button', () => {
    held = createDrag(getTab)
    held.drag.press('plex', pointer('pointerdown', 350, 300, 2))
    window.dispatchEvent(pointer('pointermove', 400, 300))

    expect(held.drag.hasMoved.value).toBe(false)
  })
})

describe('a tab being dragged', () => {
  it('is named at the pointer by its title', () => {
    held = createDrag(getTab)
    held.drag.press('plex', pointer('pointerdown', 350, 300))
    window.dispatchEvent(pointer('pointermove', 360, 300))

    expect(held.drag.label.value).toBe('Plex')
  })

  it('is named by its identity where nothing says what it is', () => {
    held = createDrag(() => undefined)
    held.drag.press('plex', pointer('pointerdown', 350, 300))
    window.dispatchEvent(pointer('pointermove', 360, 300))

    expect(held.drag.label.value).toBe('plex')
  })

  it('shows the area a drop would take, in the coordinates of the frame', () => {
    held = createDrag(getTab)
    held.drag.press('plex', pointer('pointerdown', 350, 300))
    window.dispatchEvent(pointer('pointermove', 850, 300))

    expect(held.drag.landing.value).toMatchObject({ kind: 'pane', pane: 'aside', side: 'center' })
    expect(held.drag.overlay.value).toStrictEqual({ x: 500, y: 0, width: 500, height: 600 })
  })

  it('shows nothing where it would land nowhere', () => {
    held = createDrag(getTab)
    held.drag.press('plex', pointer('pointerdown', 350, 300))
    window.dispatchEvent(pointer('pointermove', 20, 300))

    expect(held.drag.hasMoved.value).toBe(true)
    expect(held.drag.landing.value).toBeNull()
    expect(held.drag.overlay.value).toBeNull()
  })
})

describe('letting go of a tab', () => {
  it('puts it into the pane it is over', () => {
    held = createDrag(getTab)
    held.drag.press('plex', pointer('pointerdown', 350, 300))
    window.dispatchEvent(pointer('pointermove', 850, 300))
    window.dispatchEvent(pointer('pointerup', 850, 300))

    expect(tabsOf(held.workspace.value, 'aside')).toStrictEqual(['chat', 'plex'])
  })

  it('changes nothing where the press never became a drag', () => {
    held = createDrag(getTab)
    const before = held.workspace.value
    held.drag.press('plex', pointer('pointerdown', 350, 300))
    window.dispatchEvent(pointer('pointerup', 350, 300))

    expect(held.workspace.value).toBe(before)
  })
})
