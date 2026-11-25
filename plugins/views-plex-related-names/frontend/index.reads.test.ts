/**
 * How often the two views of every node of the plex read the notes: once for a node, however many
 * of its views are drawn, with the notes of a vault that answers as the core does.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { Component } from 'vue'
import type { JsonValue, NoteGraph, NoteRelation, VaultNotes } from '@numen/plugin-sdk'
import {
  createMockHostServices,
  createMockNoticeHost,
  createMockPluginContext,
} from '@numen/plugin-sdk/testing'
import { activate } from './index'

const reported = (text: string): Error => Object.assign(new Error(text), { name: 'ReportedError' })

const SETTLE_MS = 5000

const getTitle = (path: string) => path.replace(/^.*\//, '').replace(/\.md$/, '')

const createGraph = (
  path: string,
  links: Readonly<Record<string, NoteRelation>>,
  depth: number | undefined,
): NoteGraph => {
  const own = Object.entries(links).map(([one, relation]) => ({
    path: one,
    title: getTitle(one),
    relation,
    type: 'note',
  }))
  const first = own[0]
  const beyond =
    depth === 2 && first
      ? [{ ...first, path: 'notes/Beyond.md', title: 'Beyond', relatedTo: first.path }]
      : []
  return {
    focus: { path, title: getTitle(path) },
    focusType: 'note',
    related: [...own, ...beyond],
    connections: [],
  }
}

const LEAVES = Array.from({ length: 5 }, (_, index) => `notes/Leaf ${index + 1}.md`)

const setup = (initialSettings: Record<string, JsonValue> = {}) => {
  const requests: [string, number | undefined][] = []
  const links: Record<string, Record<string, NoteRelation>> = {}
  let failure: Error | undefined
  const notes = {
    listRelatedNotes: async (paths: readonly string[], depth?: 1 | 2): Promise<NoteGraph[]> =>
      paths.map((path) => {
        requests.push([path, depth])
        if (failure) throw failure
        return createGraph(path, links[path] ?? {}, depth)
      }),
  } as unknown as VaultNotes
  const notices = createMockNoticeHost()
  const context = createMockPluginContext({
    pluginId: 'md.numen.views.plex.related.names',
    initialSettings,
  })
  const host = createMockHostServices({ notes, notices })
  activate(context, host)
  const wrappers: VueWrapper[] = []
  const show = (id: string, path: string) => {
    const found = context.getRegisteredSlotViews().find((one) => one.id === id)
    const wrapper = mount(found!.component as Component, {
      props: { path },
      attachTo: document.body,
    })
    wrappers.push(wrapper)
    return wrapper
  }
  const getReads = (path?: string) =>
    path === undefined ? requests : requests.filter(([one]) => one === path)
  const fail = (error: Error) => {
    failure = error
  }
  return { context, host, links, notices, show, getReads, fail, wrappers }
}

const getNames = (wrapper: VueWrapper) =>
  wrapper.findAll('[role="button"], button').map((one) => one.text())

const cleanups: (() => void)[] = []

afterEach(() => {
  vi.useRealTimers()
  while (cleanups.length > 0) cleanups.pop()?.()
  document.body.replaceChildren()
})

const setupRig = (initialSettings: Record<string, JsonValue> = {}) => {
  const rig = setup(initialSettings)
  cleanups.push(() =>
    rig.wrappers.filter((one) => one.element.isConnected).forEach((one) => one.unmount()),
  )
  return rig
}

describe('the views of the nodes of a plex that opens', () => {
  it('mounts the parents and related views of five nodes and reads five notes, not ten', async () => {
    const { show, getReads } = setupRig()

    LEAVES.forEach((one) => {
      show('parents', one)
      show('related', one)
    })
    await flushPromises()

    expect(getReads()).toHaveLength(5)
    expect(getReads().map(([path]) => path)).toStrictEqual(LEAVES)
  })

  it('reads the notes related to each node with depth 1', async () => {
    const { show, getReads } = setupRig()

    show('parents', LEAVES[0]!)
    show('related', LEAVES[0]!)
    await flushPromises()

    expect(getReads()).toStrictEqual([[LEAVES[0], 1]])
  })

  it('starts the reads of the five nodes in the tick they mount in', async () => {
    const { show, getReads } = setupRig()

    LEAVES.forEach((one) => {
      show('parents', one)
      show('related', one)
    })
    await flushPromises()

    expect(getReads()).toHaveLength(5)
  })

  it('gives the view above and the view under a node each its own side of the one answer', async () => {
    const { show, links, getReads } = setupRig()
    links['notes/Pear rows.md'] = {
      'notes/Orchard plan.md': 'parent',
      'notes/Rootstock list.md': 'child',
    }

    const above = show('parents', 'notes/Pear rows.md')
    const below = show('related', 'notes/Pear rows.md')
    await flushPromises()

    expect(getNames(above)).toStrictEqual(['Orchard plan'])
    expect(getNames(below)).toStrictEqual(['Rootstock list'])
    expect(getReads()).toHaveLength(1)
  })

  it('reads a note once when the view under a node mounts after the view above it has its answer', async () => {
    const { show, links, getReads } = setupRig()
    links['notes/Pear rows.md'] = { 'notes/Rootstock list.md': 'child' }
    show('parents', 'notes/Pear rows.md')
    await flushPromises()

    const below = show('related', 'notes/Pear rows.md')
    await flushPromises()

    expect(getNames(below)).toStrictEqual(['Rootstock list'])
    expect(getReads()).toHaveLength(1)
  })
})

describe('the views of nodes that leave the plex', () => {
  it('keeps the answer while the other view of the node is drawn', async () => {
    const { show, getReads } = setupRig()
    const above = show('parents', LEAVES[0]!)
    show('related', LEAVES[0]!)
    await flushPromises()

    above.unmount()
    show('parents', LEAVES[0]!)
    await flushPromises()

    expect(getReads()).toHaveLength(1)
  })

  it('reads a note again for a view mounted after both of its views were unmounted', async () => {
    const { show, getReads } = setupRig()
    const above = show('parents', LEAVES[0]!)
    const below = show('related', LEAVES[0]!)
    await flushPromises()

    above.unmount()
    below.unmount()
    show('related', LEAVES[0]!)
    await flushPromises()

    expect(getReads()).toHaveLength(2)
  })

  it('releases the note a view leaves when the node shows another note', async () => {
    const { show, getReads } = setupRig()
    const below = show('related', LEAVES[0]!)
    await flushPromises()

    await below.setProps({ path: LEAVES[1]! })
    await flushPromises()
    show('related', LEAVES[0]!)
    await flushPromises()

    expect(getReads().map(([path]) => path)).toStrictEqual([LEAVES[0], LEAVES[1], LEAVES[0]])
  })

  it('reads nothing for the nodes of a plex that was closed when the vault changes', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const { show, host, getReads } = setupRig()
    const views = LEAVES.flatMap((one) => [show('parents', one), show('related', one)])
    await flushPromises()

    views.forEach((one) => one.unmount())
    host.events.emitVaultChange({})
    await vi.advanceTimersByTimeAsync(SETTLE_MS)

    expect(getReads()).toHaveLength(5)
  })
})

describe('the views of a node while the vault changes', () => {
  const setupDrawn = async () => {
    const rig = setupRig()
    rig.links['notes/Pear rows.md'] = {
      'notes/Orchard plan.md': 'parent',
      'notes/Rootstock list.md': 'child',
    }
    const above = rig.show('parents', 'notes/Pear rows.md')
    const below = rig.show('related', 'notes/Pear rows.md')
    await flushPromises()
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    return { ...rig, above, below }
  }

  const settle = async () => {
    await vi.advanceTimersByTimeAsync(SETTLE_MS)
    await flushPromises()
  }

  it('reads the note once for both views when it changed', async () => {
    const { host, getReads } = await setupDrawn()

    host.events.emitVaultChange({ paths: ['notes/Pear rows.md'] })
    await settle()

    expect(getReads()).toHaveLength(2)
  })

  it('updates the names above and under the node from that one read', async () => {
    const { host, links, above, below } = await setupDrawn()
    links['notes/Pear rows.md'] = {
      'notes/Orchard plan.md': 'parent',
      'notes/Orchard map.md': 'parent',
      'notes/Rootstock list.md': 'child',
      'notes/Compost.md': 'child',
    }

    host.events.emitVaultChange({ paths: ['notes/Pear rows.md'] })
    await settle()

    expect(getNames(above)).toStrictEqual(['Orchard plan', 'Orchard map'])
    expect(getNames(below)).toStrictEqual(['Rootstock list', 'Compost'])
  })

  it('reads the note once for both views when a related note of it changed', async () => {
    const { host, getReads } = await setupDrawn()

    host.events.emitVaultChange({ paths: ['notes/Rootstock list.md'] })
    await settle()

    expect(getReads()).toHaveLength(2)
  })

  it('reads once per node for a change that does not say which notes changed', async () => {
    const { host, show, getReads } = await setupDrawn()
    show('parents', LEAVES[0]!)
    show('related', LEAVES[0]!)
    await flushPromises()

    host.events.emitVaultChange({})
    await settle()

    expect(getReads('notes/Pear rows.md')).toHaveLength(2)
    expect(getReads(LEAVES[0])).toHaveLength(2)
  })
})

describe('the views when the read of a note fails', () => {
  it('shows one error for the two views of the node', async () => {
    const { show, fail, notices } = setupRig()
    fail(reported('the vault is closed'))

    show('parents', LEAVES[0]!)
    show('related', LEAVES[0]!)
    await flushPromises()

    expect(notices.getShown()).toStrictEqual([{ text: 'the vault is closed', severity: 'error' }])
  })

  it('shows one error for the nodes of the plex', async () => {
    const { show, fail, notices } = setupRig()
    fail(reported('the vault is closed'))

    LEAVES.forEach((one) => {
      show('parents', one)
      show('related', one)
    })
    await flushPromises()

    expect(notices.getShown()).toHaveLength(1)
  })

  it('reads the note again for a view that mounts after the failure', async () => {
    const { show, fail, getReads } = setupRig()
    fail(reported('the vault is closed'))
    show('parents', LEAVES[0]!)
    await flushPromises()

    show('related', LEAVES[0]!)
    await flushPromises()

    expect(getReads()).toHaveLength(2)
  })
})

describe('the notes the plex draws as nodes', () => {
  it('are not named by the views of the other nodes', async () => {
    const { show, links } = setupRig()
    links['notes/Pear rows.md'] = {
      'notes/Orchard plan.md': 'parent',
      'notes/Rootstock list.md': 'child',
      'notes/Compost.md': 'child',
    }
    const above = show('parents', 'notes/Pear rows.md')
    const below = show('related', 'notes/Pear rows.md')
    show('related', 'notes/Rootstock list.md')
    await flushPromises()

    expect(getNames(above)).toStrictEqual(['Orchard plan'])
    expect(getNames(below)).toStrictEqual(['Compost'])
  })

  it('are named again once their node leaves the plex', async () => {
    const { show, links } = setupRig()
    links['notes/Pear rows.md'] = { 'notes/Rootstock list.md': 'child' }
    const below = show('related', 'notes/Pear rows.md')
    const other = show('related', 'notes/Rootstock list.md')
    await flushPromises()
    expect(getNames(below)).toStrictEqual([])

    other.unmount()
    await flushPromises()

    expect(getNames(below)).toStrictEqual(['Rootstock list'])
  })

  it('keeps the names that still fit under the limit', async () => {
    const { show, links } = setupRig({ namesPerNote: 1 })
    links['notes/Pear rows.md'] = {
      'notes/Rootstock list.md': 'child',
      'notes/Compost.md': 'child',
    }
    const below = show('related', 'notes/Pear rows.md')
    show('related', 'notes/Rootstock list.md')
    await flushPromises()

    expect(getNames(below)).toStrictEqual(['Compost'])
  })
})
