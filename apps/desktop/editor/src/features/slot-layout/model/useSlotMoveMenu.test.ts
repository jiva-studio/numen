/** The right-click menu of the bar entries: where it opens, what it lists, and what a choice writes. Every plugin here is invented. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { effectScope } from 'vue'
import type { LayoutEntry, SlotLayout, SlotName } from '@numen/plugin-host'
import { useSlotMoveMenu, type SlotMoveMenuControl } from './useSlotMoveMenu'

const TIDE = 'titleBar.status|org.example.tide/status'
const REEF = 'titleBar.status|org.example.reef/stats'
const CLOCK = 'statusBar.left|org.example.tick/clock'
const TIDE_ITEM = 'org.example.tide/view/status'
const REEF_ITEM = 'org.example.reef/view/stats'
const CLOCK_ITEM = 'org.example.tick/view/clock'
const OPEN_ITEM = 'org.example.tide/command/open'

const KEYS: Record<string, string> = {
  [TIDE_ITEM]: TIDE,
  [REEF_ITEM]: REEF,
  [CLOCK_ITEM]: CLOCK,
}
const DRAWN: Partial<Record<SlotName, readonly LayoutEntry[]>> = {
  'titleBar.status': [
    { key: TIDE, title: 'Tasks' },
    { key: REEF, title: 'Index statistics' },
  ],
  'statusBar.left': [{ key: CLOCK, title: 'Clock' }],
}

const scopes: ReturnType<typeof effectScope>[] = []

const mountBars = (): HTMLElement => {
  document.body.innerHTML = `
    <div data-numen-slot="titleBar.status">
      <span id="tide" data-numen-slot-entry="${TIDE_ITEM}"><b id="inside">Tasks</b></span>
      <span id="reef" data-numen-slot-entry="${REEF_ITEM}"></span>
    </div>
    <div data-numen-slot="statusBar.left"><span id="clock" data-numen-slot-entry="${CLOCK_ITEM}"></span></div>
    <div data-numen-slot="fileTree.context"><span id="row" data-numen-slot-entry="${OPEN_ITEM}"></span></div>
    <p id="text">text</p>`
  return document.body
}

const create = (layout: SlotLayout = {}) => {
  const writeLayout = vi.fn(() => Promise.resolve())
  const scope = effectScope()
  scopes.push(scope)
  const menu = scope.run(() =>
    useSlotMoveMenu({
      findLayoutKey: (itemKey, slot) => (slot === 'fileTree.context' ? undefined : KEYS[itemKey]),
      listEntries: (slot) => DRAWN[slot] ?? [],
      getSlotTitle: (slot) => `title of ${slot}`,
      getLayout: () => layout,
      writeLayout,
      words: { moveTo: 'Move to', atTheEnd: 'At the end' },
    }),
  ) as SlotMoveMenuControl
  return { menu, writeLayout }
}

const rightClick = (id: string, at = { clientX: 40, clientY: 12 }): MouseEvent => {
  const event = new MouseEvent('contextmenu', { bubbles: true, cancelable: true, ...at })
  document.getElementById(id)?.dispatchEvent(event)
  return event
}

const findItem = (menu: SlotMoveMenuControl, ...path: string[]) => {
  let items = menu.menu.value?.items ?? []
  let found = items[0]
  for (const text of ['Move to', ...path]) {
    found = items.find((one) => one.text === text)
    items = found?.children ?? []
  }
  return found
}

afterEach(() => {
  scopes.splice(0).forEach((scope) => scope.stop())
  document.body.innerHTML = ''
})

describe('the right click on a bar entry', () => {
  it('opens the menu at the pointer and keeps the page from showing its own', () => {
    mountBars()
    const { menu } = create()

    const event = rightClick('inside')

    expect(event.defaultPrevented).toBe(true)
    expect(menu.menu.value?.at).toStrictEqual({ x: 40, y: 12 })
    expect(menu.menu.value?.items.map((one) => one.text)).toStrictEqual(['Move to'])
  })

  it('is left to the page on anything else: text and an entry of a slot the manifests fix', () => {
    mountBars()
    const { menu } = create()

    for (const id of ['text', 'row']) {
      expect(rightClick(id).defaultPrevented).toBe(false)
    }

    expect(menu.menu.value).toBeNull()
  })

  it('lists every other entry of each slot the user arranges, and not the entry itself', () => {
    mountBars()
    const { menu } = create()
    rightClick('tide')

    const status = findItem(menu, 'title of titleBar.status')
    const left = findItem(menu, 'title of statusBar.left')
    const commands = findItem(menu, 'title of titleBar.commands')

    expect(status?.children?.map((one) => one.text)).toStrictEqual([
      'Index statistics',
      'At the end',
    ])
    expect(left?.children?.map((one) => one.text)).toStrictEqual(['Clock', 'At the end'])
    expect(commands?.children?.map((one) => one.text)).toStrictEqual(['At the end'])
  })

  it('closes when it is dismissed', () => {
    mountBars()
    const { menu } = create()
    rightClick('tide')

    menu.dismiss()

    expect(menu.menu.value).toBeNull()
  })
})

describe('a choice of the menu', () => {
  it('writes both lists of a move into another slot in one write, and closes the menu', () => {
    mountBars()
    const { menu, writeLayout } = create()
    rightClick('tide')

    menu.choose(findItem(menu, 'title of statusBar.left', 'Clock')?.id ?? '')

    expect(writeLayout).toHaveBeenCalledTimes(1)
    expect(writeLayout).toHaveBeenCalledWith({
      'titleBar.status': [REEF],
      'statusBar.left': [TIDE, CLOCK],
    })
    expect(menu.menu.value).toBeNull()
  })

  it('puts the entry last for At the end', () => {
    mountBars()
    const { menu, writeLayout } = create()
    rightClick('tide')

    menu.choose(findItem(menu, 'title of titleBar.status', 'At the end')?.id ?? '')

    expect(writeLayout).toHaveBeenCalledWith({ 'titleBar.status': [REEF, TIDE] })
  })

  it('writes nothing for a move that changes nothing, and closes the menu', () => {
    mountBars()
    const { menu, writeLayout } = create()
    rightClick('tide')

    menu.choose(findItem(menu, 'title of titleBar.status', 'Index statistics')?.id ?? '')

    expect(writeLayout).not.toHaveBeenCalled()
    expect(menu.menu.value).toBeNull()
  })

  it('writes nothing for an item that is no move, and keeps the menu open', () => {
    mountBars()
    const { menu, writeLayout } = create()
    rightClick('tide')

    menu.choose(findItem(menu)?.id ?? '')
    menu.choose('unknown')

    expect(writeLayout).not.toHaveBeenCalled()
    expect(menu.menu.value).not.toBeNull()
  })

  it('writes nothing when the menu is not open', () => {
    mountBars()
    const { menu, writeLayout } = create()

    menu.choose('end:statusBar.left')

    expect(writeLayout).not.toHaveBeenCalled()
  })
})

describe('the menu after its scope stops', () => {
  it('does not open', () => {
    mountBars()
    const { menu } = create()
    scopes.splice(0).forEach((scope) => scope.stop())

    rightClick('tide')

    expect(menu.menu.value).toBeNull()
  })
})
