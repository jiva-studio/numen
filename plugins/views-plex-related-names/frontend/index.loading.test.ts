/**
 * What the two views tell the plex about their data: a view is loading until the notes related to its
 * node have been read, and the owner of the slot waits for it, with the readiness of the SDK.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { defineComponent, h, provide, type Component } from 'vue'
import {
  SLOT_LOADING_KEY,
  createSlotLoading,
  type JsonValue,
  type NoteGraph,
  type SlotLoadingEntry,
  type VaultNotes,
} from '@numen/plugin-sdk'
import {
  createMockHostServices,
  createMockNoticeHost,
  createMockPluginContext,
} from '@numen/plugin-sdk/testing'
import { activate } from './index'

const reported = (text: string): Error => Object.assign(new Error(text), { name: 'ReportedError' })

const PEAR = 'notes/Pear rows.md'
const QUINCE = 'notes/Quince.md'
const ADVANCE_MS = 5000
const ENTRY: SlotLoadingEntry = {
  key: 'md.numen.views.plex.related.names/view/related',
  pluginId: 'md.numen.views.plex.related.names',
}

interface Pending {
  readonly path: string
  resolve(): void
  reject(error: Error): void
}

interface Call {
  readonly paths: readonly string[]
  resolve(graphs: NoteGraph[]): void
  reject(error: Error): void
}

const createGraph = (path: string): NoteGraph => ({
  focus: { path, title: path },
  focusType: 'note',
  related: [
    { path: 'notes/Rootstock list.md', title: 'Rootstock list', relation: 'child', type: 'note' },
  ],
  connections: [],
})

const setup = (initialSettings: Record<string, JsonValue> = {}) => {
  const pending: Pending[] = []
  const notes = {
    listRelatedNotes: (paths: readonly string[]) =>
      new Promise<NoteGraph[]>((resolve, reject) => {
        const call: Call = { paths, resolve, reject }
        for (const path of paths) {
          pending.push({
            path,
            resolve: () => call.resolve(call.paths.map(createGraph)),
            reject: call.reject,
          })
        }
      }),
  } as unknown as VaultNotes
  const notices = createMockNoticeHost()
  const context = createMockPluginContext({
    pluginId: 'md.numen.views.plex.related.names',
    initialSettings,
  })
  const host = createMockHostServices({ notes, notices })
  activate(context, host)

  const loading = createSlotLoading({
    clock: { now: () => 0, schedule: () => 0, cancel: () => undefined },
  })
  const tokens = { begun: 0, released: 0 }
  const wrappers: VueWrapper[] = []
  const show = (id: string, path: string) => {
    const found = context.getRegisteredSlotViews().find((one) => one.id === id)
    const view = found!.component as Component
    const owner = defineComponent({
      setup() {
        provide(SLOT_LOADING_KEY, {
          begin: () => {
            tokens.begun += 1
            const release = loading.begin(ENTRY)
            return () => {
              tokens.released += 1
              release()
            }
          },
        })
        return () => h(view, { path })
      },
    })
    const wrapper = mount(owner, { attachTo: document.body })
    wrappers.push(wrapper)
    return wrapper
  }
  const getReads = (path?: string) =>
    path === undefined ? pending : pending.filter((one) => one.path === path)
  const answer = async (path: string) => {
    await flushPromises()
    getReads(path).forEach((one) => one.resolve())
    await flushPromises()
  }
  return { context, host, loading, tokens, notices, show, getReads, answer, wrappers }
}

const rigs: ReturnType<typeof setup>[] = []

const setupRig = (initialSettings: Record<string, JsonValue> = {}) => {
  const rig = setup(initialSettings)
  rigs.push(rig)
  return rig
}

afterEach(() => {
  vi.useRealTimers()
  for (const rig of rigs.splice(0)) {
    rig.wrappers.filter((one) => one.element.isConnected).forEach((one) => one.unmount())
  }
  document.body.replaceChildren()
})

describe('the views while the notes of their node are read', () => {
  it('keeps the slot not ready while the view under a node waits for its notes', () => {
    const { show, loading } = setupRig()

    show('related', PEAR)

    expect(loading.isReady.value).toBe(false)
  })

  it('keeps the slot not ready while the view above a node waits for its notes', () => {
    const { show, loading } = setupRig()

    show('parents', PEAR)

    expect(loading.isReady.value).toBe(false)
  })

  it('makes the slot ready once the notes of the node have been read, for the view under it', async () => {
    const { show, loading, answer } = setupRig()
    show('related', PEAR)

    await answer(PEAR)

    expect(loading.isReady.value).toBe(true)
  })

  it('makes the slot ready once the notes of the node have been read, for the view above it', async () => {
    const { show, loading, answer } = setupRig()
    show('parents', PEAR)

    await answer(PEAR)

    expect(loading.isReady.value).toBe(true)
  })

  it('waits for the notes of every node that is drawn, which are read in one call', async () => {
    const { show, loading, answer, getReads } = setupRig()
    show('related', PEAR)
    show('related', QUINCE)
    await flushPromises()

    expect(loading.isReady.value).toBe(false)
    expect(getReads().map((one) => one.path)).toStrictEqual([PEAR, QUINCE])

    await answer(PEAR)
    expect(loading.isReady.value).toBe(true)
  })

  it('draws the names in the view once the slot is ready', async () => {
    const { show, answer } = setupRig()
    const wrapper = show('related', PEAR)

    await answer(PEAR)

    expect(wrapper.findAll('[role="button"], button').map((one) => one.text())).toStrictEqual([
      'Rootstock list',
    ])
  })

  it('stays ready when the vault changes and the notes are read again', async () => {
    const { show, loading, answer, host } = setupRig()
    show('related', PEAR)
    await answer(PEAR)
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })

    host.events.emitVaultChange({ paths: [PEAR] })
    await vi.advanceTimersByTimeAsync(ADVANCE_MS)

    expect(loading.isReady.value).toBe(true)
  })
})

describe('the views when the read of the notes fails', () => {
  it('makes the slot ready, because the view has nothing more to wait for', async () => {
    const { show, loading, getReads } = setupRig()
    show('related', PEAR)

    await flushPromises()
    getReads(PEAR)[0]?.reject(reported('the vault is closed'))
    await flushPromises()

    expect(loading.isReady.value).toBe(true)
  })

  it('reports the failure once for the two views of the node', async () => {
    const { show, getReads, notices } = setupRig()
    show('parents', PEAR)
    show('related', PEAR)

    await flushPromises()
    getReads(PEAR)[0]?.reject(reported('the vault is closed'))
    await flushPromises()

    expect(notices.getShown()).toStrictEqual([{ text: 'the vault is closed', severity: 'error' }])
  })

  it('makes the slot ready for both views of the node', async () => {
    const { show, loading, getReads } = setupRig()
    show('parents', PEAR)
    show('related', PEAR)

    await flushPromises()
    getReads(PEAR)[0]?.reject(reported('the vault is closed'))
    await flushPromises()

    expect(loading.isReady.value).toBe(true)
  })
})

describe('the two views of one node', () => {
  it('read the notes once and begin one token each', async () => {
    const { show, tokens, getReads } = setupRig()

    show('parents', PEAR)
    show('related', PEAR)
    await flushPromises()

    expect(getReads(PEAR)).toHaveLength(1)
    expect(tokens.begun).toBe(2)
  })

  it('release their own token each when the one answer arrives', async () => {
    const { show, tokens, answer } = setupRig()
    show('parents', PEAR)
    show('related', PEAR)

    await answer(PEAR)

    expect(tokens.released).toBe(2)
  })

  it('release a token once, however often the notes are read afterwards', async () => {
    const { show, tokens, answer, host } = setupRig()
    show('parents', PEAR)
    show('related', PEAR)
    await answer(PEAR)
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })

    host.events.emitVaultChange({ paths: [PEAR] })
    await vi.advanceTimersByTimeAsync(ADVANCE_MS)

    expect(tokens.released).toBe(2)
  })

  it('finish the view that shows a node without a path at once', () => {
    const { show, loading, tokens } = setupRig()

    show('related', '')

    expect(tokens.begun).toBe(1)
    expect(loading.isReady.value).toBe(true)
  })
})

describe('a view that leaves before its notes are read', () => {
  it('releases its token when it is unmounted', () => {
    const { show, tokens, loading } = setupRig()
    const wrapper = show('related', PEAR)

    wrapper.unmount()

    expect(tokens.released).toBe(tokens.begun)
    expect(loading.isReady.value).toBe(true)
  })

  it('keeps the token of the other view of the node until the notes are read', async () => {
    const { show, tokens, answer } = setupRig()
    const above = show('parents', PEAR)
    show('related', PEAR)

    above.unmount()
    expect(tokens.released).toBe(1)

    await answer(PEAR)
    expect(tokens.released).toBe(2)
  })

  it('leaves no token pending after the view is unmounted before the answer', async () => {
    const { show, tokens, loading, answer } = setupRig()
    show('related', PEAR).unmount()

    await answer(PEAR)

    expect(tokens.released).toBe(tokens.begun)
    expect(loading.isReady.value).toBe(true)
  })

  it('releases the token once when the view is unmounted after it finished', async () => {
    const { show, tokens, answer } = setupRig()
    const wrapper = show('related', PEAR)
    await answer(PEAR)

    wrapper.unmount()

    expect(tokens.released).toBe(1)
  })
})
