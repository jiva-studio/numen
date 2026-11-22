import { afterEach, describe, expect, it } from 'vitest'
import { commands } from 'vitest/browser'
import { createApp, h, shallowRef } from 'vue'
import '@numen/ui/styles.css'
import type { JsonValue } from '@/shared/api/json'
import { findDiscontinuities, startProbe } from '../../testing/frameProbe'
import PluginSlots from './PluginSlots.vue'

const groups = [
  {
    slot: 'plexGraph.nodeBelow',
    title: 'Below a node',
    contributors: ['one', 'two', 'three', 'four'].map((label) => ({
      key: `a/view/${label}`,
      label,
      pluginName: 'Alpha',
    })),
  },
]

const withSeparator = [
  {
    slot: 'titleBar.commands',
    title: 'Actions',
    contributors: [
      ['one', 'a'],
      ['two', 'a'],
      ['three', 'b'],
      ['four', 'b'],
    ].map(([label, group]) => ({
      key: `a/command/${label}`,
      label: label!,
      pluginName: 'Alpha',
      group,
    })),
  },
]

let host: HTMLElement | null = null
let stored: JsonValue | undefined
let renders = 0

afterEach(() => {
  host?.remove()
  host = null
})

function mountSlots(shown: typeof groups = groups): HTMLElement {
  host = document.createElement('div')
  host.style.width = '420px'
  document.body.append(host)
  renders = 0
  stored = undefined
  const layout = shallowRef<JsonValue | undefined>(undefined)
  const app = createApp({
    render: () =>
      h(PluginSlots, {
        groups: shown,
        layout: layout.value,
        onChange: (next: JsonValue) => {
          layout.value = next
          stored = next
        },
      }),
  })
  app.mixin({
    beforeUpdate() {
      if (this.$options.__name === 'PluginSlots') renders += 1
    },
  })
  app.mount(host)
  return host
}

const rowsOf = (root: HTMLElement): HTMLElement[] =>
  Array.from(root.querySelectorAll<HTMLElement>('.plugin-slots__row'))

const nextFrame = (): Promise<void> => new Promise((done) => requestAnimationFrame(() => done()))

async function grab(root: HTMLElement, at = 0) {
  const rows = rowsOf(root)
  const grip = rows[at]!.querySelector('button')!.getBoundingClientRect()
  const x = grip.left + grip.width / 2
  const y = grip.top + grip.height / 2
  const pitch = rows[1]!.getBoundingClientRect().top - rows[0]!.getBoundingClientRect().top
  await commands.pointer('move', { x, y })
  await commands.pointer('down')
  const moveTo = async (offset: number): Promise<void> => {
    await commands.pointer('move', { x, y: y + offset })
    await nextFrame()
  }
  return { pitch, moveTo }
}

describe('the slots of a plugin dragged by a real pointer', () => {
  it('renders the list once for each place the dragged row takes', async () => {
    const root = mountSlots()
    await nextFrame()
    const { moveTo } = await grab(root)
    await moveTo(6)
    renders = 0
    for (let offset = 8; offset <= 60; offset += 2) await moveTo(offset)
    await commands.pointer('up')

    expect(renders).toBeLessThanOrEqual(3)
  })

  it('leaves every neighbour untouched by the moves of the dragged row within one place', async () => {
    const root = mountSlots()
    await nextFrame()
    const { moveTo } = await grab(root)
    await moveTo(6)
    const neighbours = rowsOf(root).slice(1)
    let touched = 0
    const watcher = new MutationObserver((records) => {
      touched += records.length
    })
    neighbours.forEach((one) =>
      watcher.observe(one, { attributes: true, subtree: true, childList: true }),
    )
    for (let offset = 7; offset <= 12; offset++) await moveTo(offset)
    watcher.disconnect()
    await commands.pointer('up')

    expect(touched).toBe(0)
  })

  it('start no transition of a neighbour while the pointer trembles on the boundary of two rows', async () => {
    const root = mountSlots()
    await nextFrame()
    const { pitch, moveTo } = await grab(root)
    await moveTo(pitch / 2 - 6)
    let runs = 0
    root.addEventListener('transitionrun', () => (runs += 1))
    for (let step = 0; step < 20; step++) await moveTo(pitch / 2 + (step % 2 === 0 ? 1 : -1))
    await commands.pointer('up')

    expect(runs).toBeLessThanOrEqual(1)
  })

  it('start one transition of each neighbour while the pointer goes down across the rows', async () => {
    const root = mountSlots()
    await nextFrame()
    const { moveTo } = await grab(root)
    const runs = new Map<EventTarget, number>()
    root.addEventListener(
      'transitionrun',
      (event) =>
        event.target instanceof HTMLLIElement &&
        event.propertyName === 'transform' &&
        runs.set(event.target, (runs.get(event.target) ?? 0) + 1),
    )
    for (let offset = 2; offset <= 140; offset += 2) await moveTo(offset)
    const seen = [...runs.values()]
    await commands.pointer('up')

    expect(seen).toStrictEqual([1, 1, 1])
  })

  it('select no text while a row is dragged', async () => {
    const root = mountSlots()
    await nextFrame()
    const { moveTo } = await grab(root)
    for (let offset = 2; offset <= 80; offset += 4) await moveTo(offset)
    const selected = window.getSelection()?.toString() ?? ''
    await commands.pointer('up')

    expect(selected).toBe('')
  })

  it.each([
    { name: 'down across two rows', from: 0, by: 2 },
    { name: 'up across two rows', from: 2, by: -2 },
  ])(
    'show no jump, flash or replaced row on any frame of a drag $name and its drop',
    async ({ from, by }) => {
      const root = mountSlots()
      await nextFrame()
      const grip = rowsOf(root)[from]!.querySelector('button')!.getBoundingClientRect()
      const x = grip.left + grip.width / 2
      const y = grip.top + grip.height / 2
      const pitch =
        rowsOf(root)[1]!.getBoundingClientRect().top - rowsOf(root)[0]!.getBoundingClientRect().top
      const probe = startProbe(root)
      await commands.pointer('move', { x, y })
      await commands.pointer('down')
      const sign = Math.sign(by)
      for (let offset = 1; offset <= Math.abs(by) * pitch + 10; offset += 2) {
        await commands.pointer('move', { x, y: y + sign * offset })
        await nextFrame()
      }
      await commands.pointer('up')
      for (let frame = 0; frame < 40; frame++) await nextFrame()

      expect(findDiscontinuities(probe.stop())).toStrictEqual([])
    },
  )
})

describe('the slots of a plugin with a separator between two groups', () => {
  const labelsOf = (root: HTMLElement): string[] =>
    rowsOf(root).map((one) => one.querySelector('.plugin-slots__label')?.textContent ?? '|')

  it('draw the separator as tall as a row', async () => {
    const root = mountSlots(withSeparator)
    await nextFrame()
    const heights = rowsOf(root).map((one) => one.getBoundingClientRect().height)

    expect(new Set(heights).size).toBe(1)
  })

  it('move a row into the other group when it is dragged across the separator', async () => {
    const root = mountSlots(withSeparator)
    await nextFrame()
    const { pitch, moveTo } = await grab(root, 1)
    for (let offset = 2; offset <= pitch * 1.6; offset += 2) await moveTo(offset)
    await commands.pointer('up')
    for (let frame = 0; frame < 40; frame++) await nextFrame()

    expect(labelsOf(root)).toStrictEqual(['one', '|', 'two', 'three', 'four'])
    expect(stored).toStrictEqual({
      [withSeparator[0]!.slot]: [
        { id: 'a/command/one' },
        { id: '---' },
        { id: 'a/command/two' },
        { id: 'a/command/three' },
        { id: 'a/command/four' },
      ],
    })
  })

  it('move the separator itself like a row, and store it at its new place', async () => {
    const root = mountSlots(withSeparator)
    await nextFrame()
    const { pitch, moveTo } = await grab(root, 2)
    for (let offset = 2; offset <= pitch * 1.6; offset += 2) await moveTo(offset)
    await commands.pointer('up')
    for (let frame = 0; frame < 40; frame++) await nextFrame()

    expect(labelsOf(root)).toStrictEqual(['one', 'two', 'three', '|', 'four'])
    expect(stored).toStrictEqual({
      [withSeparator[0]!.slot]: [
        { id: 'a/command/one' },
        { id: 'a/command/two' },
        { id: 'a/command/three' },
        { id: '---' },
        { id: 'a/command/four' },
      ],
    })
  })

  it('show no jump, flash or replaced row on any frame of a drag of the separator', async () => {
    const root = mountSlots(withSeparator)
    await nextFrame()
    const probe = startProbe(root)
    const { pitch, moveTo } = await grab(root, 2)
    for (let offset = 1; offset <= pitch * 1.6; offset += 2) await moveTo(offset)
    await commands.pointer('up')
    for (let frame = 0; frame < 40; frame++) await nextFrame()

    expect(findDiscontinuities(probe.stop())).toStrictEqual([])
  })

  it('show no jump, flash or replaced row on any frame of a drag across the separator', async () => {
    const root = mountSlots(withSeparator)
    await nextFrame()
    const probe = startProbe(root)
    const { pitch, moveTo } = await grab(root, 1)
    for (let offset = 1; offset <= pitch * 2 + 10; offset += 2) await moveTo(offset)
    await commands.pointer('up')
    for (let frame = 0; frame < 40; frame++) await nextFrame()

    expect(findDiscontinuities(probe.stop())).toStrictEqual([])
  })
})
