/**
 * The two views of the plugin as the plex draws them: the parents of a note above its node, and
 * its children and other related notes under it, read from a vault that answers like the core.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { Component } from 'vue'
import type { JsonValue, NoteGraph, NoteRelation, RelatedNote, VaultNotes } from '@numen/plugin-sdk'
import {
  createMockHostServices,
  createMockPluginContext,
  type MockCommandsHost,
} from '@numen/plugin-sdk/testing'
import { activate } from './index'

interface Neighbor {
  readonly path: string
  readonly relation: NoteRelation
}

const getTitle = (path: string) => path.replace(/^.*\//, '').replace(/\.md$/, '')

const createVault = (neighbors: Readonly<Record<string, readonly Neighbor[]>>) => {
  const requests: [string, number | undefined][] = []
  const toRelated = (one: Neighbor, relatedTo?: string): RelatedNote => ({
    path: one.path,
    title: getTitle(one.path),
    relation: one.relation,
    type: 'note',
    ...(relatedTo ? { relatedTo } : {}),
  })
  const notes = {
    listRelatedNotes: async (path: string, depth?: 1 | 2): Promise<NoteGraph> => {
      requests.push([path, depth])
      const own = neighbors[path] ?? []
      const beyond =
        depth === 2
          ? own.flatMap((near) =>
              (neighbors[near.path] ?? [])
                .filter((far) => far.path !== path)
                .map((far) => toRelated(far, near.path)),
            )
          : []
      return {
        focus: { path, title: getTitle(path) },
        focusType: 'note',
        related: [...own.map((one) => toRelated(one)), ...beyond],
        connections: [],
      }
    },
  } as unknown as VaultNotes
  return { notes, requests }
}

const PEAR = 'notes/Pear rows.md'

const ORCHARD: Record<string, readonly Neighbor[]> = {
  [PEAR]: [
    { path: 'notes/Orchard plan.md', relation: 'parent' },
    { path: 'notes/Garden plan.md', relation: 'parent' },
    { path: 'notes/Rootstock list.md', relation: 'child' },
    { path: 'notes/Grafting.md', relation: 'jump' },
    { path: 'notes/Quince.md', relation: 'sibling' },
    { path: 'notes/Pollination.md', relation: 'reference' },
  ],
}

const wrappers: VueWrapper[] = []

const setup = (
  neighbors: Readonly<Record<string, readonly Neighbor[]>> = ORCHARD,
  initialSettings: Record<string, JsonValue> = {},
) => {
  const vault = createVault(neighbors)
  const context = createMockPluginContext({
    pluginId: 'md.numen.views.plex.relatedNames',
    initialSettings,
  })
  const host = createMockHostServices({ notes: vault.notes })
  activate(context, host)
  const getView = (id: string): Component => {
    const found = context.getRegisteredSlotViews().find((one) => one.id === id)
    expect(found, `the view ${id} is registered`).toBeDefined()
    return found!.component as Component
  }
  const show = async (id: string, path: string) => {
    const wrapper = mount(getView(id), { props: { path }, attachTo: document.body })
    wrappers.push(wrapper)
    await flushPromises()
    return wrapper
  }
  const getRuns = () => (host.ui.commands as MockCommandsHost).getRuns()
  return { context, host, vault, show, getRuns }
}

const getNames = (wrapper: VueWrapper) =>
  wrapper.findAll('[role="button"], button').map((one) => one.text())

afterEach(() => {
  while (wrappers.length > 0) wrappers.pop()?.unmount()
  document.body.replaceChildren()
})

describe('the plugin', () => {
  it('registers the two views its manifest places in the plex', () => {
    const { context } = setup()

    expect(
      context
        .getRegisteredSlotViews()
        .map((one) => one.id)
        .sort(),
    ).toStrictEqual(['parents', 'related'])
  })
})

describe('the view above a node', () => {
  it('names the parents of the note, in the order the vault lists them', async () => {
    const { show } = setup()

    expect(getNames(await show('parents', PEAR))).toStrictEqual(['Orchard plan', 'Garden plan'])
  })

  it('names no child, jump, sibling or reference', async () => {
    const { show } = setup()
    const names = getNames(await show('parents', PEAR))

    expect(names).not.toContain('Rootstock list')
    expect(names).not.toContain('Grafting')
    expect(names).not.toContain('Quince')
    expect(names).not.toContain('Pollination')
  })

  it('draws nothing for a note with no parent', async () => {
    const { show } = setup({ [PEAR]: [{ path: 'notes/Rootstock list.md', relation: 'child' }] })

    expect(getNames(await show('parents', PEAR))).toStrictEqual([])
  })

  it('gives each name a label with its title and the title of the note', async () => {
    const { show } = setup()
    const labels = (await show('parents', PEAR))
      .findAll('[role="button"], button')
      .map((one) => one.attributes('aria-label'))

    expect(labels).toStrictEqual([
      'Orchard plan, name under Pear rows',
      'Garden plan, name under Pear rows',
    ])
  })
})

describe('the view under a node', () => {
  it('names the children, then the jumps, then the references of the note', async () => {
    const { show } = setup()

    expect(getNames(await show('related', PEAR))).toStrictEqual([
      'Rootstock list',
      'Grafting',
      'Pollination',
    ])
  })

  it('names no parent and no sibling', async () => {
    const { show } = setup()
    const names = getNames(await show('related', PEAR))

    expect(names).not.toContain('Orchard plan')
    expect(names).not.toContain('Quince')
  })

  it('names a note once where it is related in several ways, in the first way of the order', async () => {
    const { show } = setup({
      [PEAR]: [
        { path: 'notes/Pollination.md', relation: 'reference' },
        { path: 'notes/Pollination.md', relation: 'child' },
        { path: 'notes/Grafting.md', relation: 'jump' },
      ],
    })

    expect(getNames(await show('related', PEAR))).toStrictEqual(['Pollination', 'Grafting'])
  })

  it('reads the notes of the one it is given, and no note beyond them', async () => {
    const { show } = setup({
      ...ORCHARD,
      'notes/Rootstock list.md': [{ path: 'notes/Clones.md', relation: 'child' }],
    })

    expect(getNames(await show('related', PEAR))).not.toContain('Clones')
  })

  it('reads the notes related to the note with depth 2', async () => {
    const { show, vault } = setup()

    await show('related', PEAR)

    expect(vault.requests).toContainEqual([PEAR, 2])
    expect(vault.requests.every(([path]) => path === PEAR)).toBe(true)
  })

  it('draws the names of the new note when the node shows another note', async () => {
    const { show } = setup({
      ...ORCHARD,
      'notes/Quince.md': [{ path: 'notes/Jam.md', relation: 'child' }],
    })
    const wrapper = await show('related', PEAR)

    await wrapper.setProps({ path: 'notes/Quince.md' })
    await flushPromises()

    expect(getNames(wrapper)).toStrictEqual(['Jam'])
  })

  it('gives each name a button label with its title and the title of the note', async () => {
    const { show } = setup()
    const labels = (await show('related', PEAR))
      .findAll('[role="button"], button')
      .map((one) => one.attributes('aria-label'))

    expect(labels).toStrictEqual([
      'Rootstock list, name under Pear rows',
      'Grafting, name under Pear rows',
      'Pollination, name under Pear rows',
    ])
  })
})

describe('choosing a name', () => {
  it('shows its note in the plex, as the command of the window does', async () => {
    const { show, getRuns } = setup()
    const wrapper = await show('related', PEAR)

    await wrapper.get('[aria-label="Rootstock list, name under Pear rows"]').trigger('click')

    expect(getRuns()).toHaveLength(1)
    expect(getRuns()[0]).toMatchObject({
      command: 'app:file.reveal',
      target: { path: 'notes/Rootstock list.md' },
      values: { kind: 'plex' },
    })
  })

  it('shows the note of a parent that is chosen above a node', async () => {
    const { show, getRuns } = setup()
    const wrapper = await show('parents', PEAR)

    await wrapper.get('[aria-label="Garden plan, name under Pear rows"]').trigger('click')

    expect(getRuns().map((one) => [one.command, one.target?.path])).toStrictEqual([
      ['app:file.reveal', 'notes/Garden plan.md'],
    ])
  })

  it('runs nothing until a name is chosen', async () => {
    const { show, getRuns } = setup()

    await show('related', PEAR)

    expect(getRuns()).toStrictEqual([])
  })
})

describe('the setting Show notes related to related notes', () => {
  it('draws no name in either view while it is off', async () => {
    const { show } = setup(ORCHARD, { showNames: false })

    expect(getNames(await show('parents', PEAR))).toStrictEqual([])
    expect(getNames(await show('related', PEAR))).toStrictEqual([])
  })

  it('takes the names away when it is turned off, and brings them back when it is turned on', async () => {
    const { context, show } = setup()
    const wrapper = await show('related', PEAR)

    context.setSetting('showNames', false)
    await flushPromises()
    expect(getNames(wrapper)).toStrictEqual([])

    context.setSetting('showNames', true)
    await flushPromises()
    expect(getNames(wrapper)).toStrictEqual(['Rootstock list', 'Grafting', 'Pollination'])
  })
})

describe('the setting Names per note', () => {
  const MANY: Record<string, readonly Neighbor[]> = {
    [PEAR]: Array.from({ length: 8 }, (_, index) => ({
      path: `notes/Leaf ${index + 1}.md`,
      relation: 'child' as const,
    })),
  }

  it('shows six names where nothing is set', async () => {
    const { show } = setup(MANY)

    expect(getNames(await show('related', PEAR))).toHaveLength(6)
  })

  it('cuts the names to the number, keeping the first', async () => {
    const { show } = setup(MANY, { namesPerNote: 2 })

    expect(getNames(await show('related', PEAR))).toStrictEqual(['Leaf 1', 'Leaf 2'])
  })

  it('shows more names when the number is raised', async () => {
    const { context, show } = setup(MANY, { namesPerNote: 2 })
    const wrapper = await show('related', PEAR)

    context.setSetting('namesPerNote', 4)
    await flushPromises()

    expect(getNames(wrapper)).toStrictEqual(['Leaf 1', 'Leaf 2', 'Leaf 3', 'Leaf 4'])
  })

  it('shows at most twelve names', async () => {
    const many = {
      [PEAR]: Array.from({ length: 20 }, (_, index) => ({
        path: `notes/Leaf ${index + 1}.md`,
        relation: 'child' as const,
      })),
    }
    const { show } = setup(many, { namesPerNote: 500 })

    expect(getNames(await show('related', PEAR))).toHaveLength(12)
  })
})
