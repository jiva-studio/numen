/**
 * The end of each row of a files tab, drawn from the views the plugins place in the slot
 * fileTree.rowEnd.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import type { SlotItem, SlotView } from '@numen/plugin-sdk'
import type { Entry } from '../types'
import FilesTab from './FilesTab.vue'
import { useFilesTab } from '../model/useFilesTab'
import { createFileTree, ROOT } from '../model/useFileTree'
import { renderDeps, ROW_MENU } from '../fixtures/host'

enableAutoUnmount(afterEach)

const file = (path: string, overrides: Partial<Entry> = {}): Entry => ({
  path,
  name: path.split('/').pop() ?? path,
  isFolder: false,
  isNote: true,
  type: 'note',
  ...overrides,
})

const FOLDERS: Record<string, readonly Entry[]> = {
  [ROOT]: [
    file('physics', { isFolder: true, isNote: false }),
    file('Entropy.md'),
    file('Cover.png', { isNote: false }),
  ],
  physics: [file('physics/Kelvin.md')],
}

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const createView = (name: string): SlotView => ({
  kind: 'view',
  key: `md.numen.test/view/${name}`,
  pluginId: 'md.numen.test',
  view: name,
  component: defineComponent({
    props: { path: { type: String, required: true } },
    setup: (props) => () => h('b', { [`data-${name}`]: props.path }, `${name} of ${props.path}`),
  }),
})

/** The slot as its owner watches it: the handler gets the entries at once and again for each change. */
const createRowEnd = (entries: readonly SlotItem[]) => {
  const handlers = new Set<(entries: readonly SlotItem[]) => void>()
  return {
    watchRowEntries: (handler: (entries: readonly SlotItem[]) => void) => {
      handlers.add(handler)
      handler(entries)
      return { dispose: () => handlers.delete(handler) }
    },
    set: (next: readonly SlotItem[]) => handlers.forEach((handler) => handler(next)),
  }
}

const mountFiles = async (entries: readonly SlotItem[]) => {
  const rowEnd = createRowEnd(entries)
  const list = createFileTree({ list: async (folder: string) => FOLDERS[folder] ?? [] })
  const tab = useFilesTab(list, {
    window: {
      ...renderDeps,
      openEntry: () => undefined,
      setDraggedPaths: () => undefined,
      showError: () => undefined,
      watchRowEntries: rowEnd.watchRowEntries,
    },
    menu: { listItems: async () => [], getMenuEntries: () => ROW_MENU, runCommand: async () => '' },
    transfer: {
      movePath: async () => undefined,
      canDropData: () => false,
      dropData: async () => [],
    },
  })
  await list.folders.openFolder(ROOT)
  await list.folders.openFolder('physics')
  const window = mount(FilesTab, { props: { state: tab }, attachTo: document.body })
  await settle()
  return { window, rowEnd }
}

const getDrawn = (window: Awaited<ReturnType<typeof mountFiles>>['window'], name: string) =>
  window.findAll(`[data-tree-row] [data-${name}]`).map((one) => one.attributes(`data-${name}`))

describe('the end of a row of a files tab', () => {
  it('draws the view a plugin placed in the slot, given the path of each row', async () => {
    const { window } = await mountFiles([createView('tags')])

    expect(getDrawn(window, 'tags')).toStrictEqual([
      'physics',
      'physics/Kelvin.md',
      'Entropy.md',
      'Cover.png',
    ])
  })

  it('draws the view inside the row it belongs to', async () => {
    const { window } = await mountFiles([createView('tags')])

    const row = window.get('[data-tree-row="Entropy.md"]')
    expect(row.get('[data-tags]').text()).toBe('tags of Entropy.md')
  })

  it('draws one view for each entry of the slot, in the order of the slot', async () => {
    const { window } = await mountFiles([createView('tags'), createView('size')])

    const row = window.get('[data-tree-row="Entropy.md"]')
    expect(row.findAll('b').map((one) => one.text())).toStrictEqual([
      'tags of Entropy.md',
      'size of Entropy.md',
    ])
  })

  it('draws nothing at the end of a row where the slot holds no entry', async () => {
    const { window } = await mountFiles([])

    expect(getDrawn(window, 'tags')).toStrictEqual([])
  })

  it('draws the entries again when the slot changes', async () => {
    const { window, rowEnd } = await mountFiles([createView('tags')])

    rowEnd.set([createView('size')])
    await settle()

    expect(getDrawn(window, 'tags')).toStrictEqual([])
    expect(getDrawn(window, 'size')).toHaveLength(4)
  })
})
