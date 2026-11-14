/**
 * The ids of tab kinds and readers belong to one enabled plugin: a second registration of an id is
 * refused, and disposing the owner frees the id.
 */
import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginHost } from './host'
import type { PluginHostDeps } from './types'
import { STUB_COMPONENT } from '../testing/stubComponent'

const createDeps = (): PluginHostDeps => ({
  target: 'desktop.editor',
  registerKind: vi.fn(() => vi.fn()),
  registerExtensionReader: vi.fn(() => vi.fn()),
  openTab: vi.fn(async () => 'tab-1'),
})

const declaring = (pluginId: string) =>
  createMockManifest(pluginId, {
    permissions: ['vault:read'],
    readers: ['pdf', 'reader:pdf'],
    tabKinds: ['deck', 'reader:pdf'],
  })

function createTwoPlugins() {
  const deps = createDeps()
  const host = createPluginHost(deps)
  const owner = host.lifecycle.createContext('md.example.owner', declaring('md.example.owner'))
  const rival = host.lifecycle.createContext('md.example.rival', declaring('md.example.rival'))
  return { deps, host, owner, rival }
}

describe('a tab kind another plugin registers', () => {
  it('is refused, naming both plugins and the kind, and the first plugin keeps it', () => {
    const { deps, host, owner, rival } = createTwoPlugins()
    owner.plugin.register.tabKind({ kind: 'deck', title: 'Owner deck', component: STUB_COMPONENT })
    const registered = vi.mocked(deps.registerKind).mock.calls.length

    expect(() =>
      rival.plugin.register.tabKind({
        kind: 'deck',
        title: 'Rival deck',
        component: STUB_COMPONENT,
      }),
    ).toThrowError(
      'Plugin "md.example.rival" registers the tab kind "deck", which plugin "md.example.owner" already registers',
    )

    expect(vi.mocked(deps.registerKind).mock.calls).toHaveLength(registered)
    expect(host.registered.getRegisteredTabKinds().map((one) => one.title)).toStrictEqual([
      'Owner deck',
    ])
  })

  it('is refused for the plugin that registers it twice', () => {
    const { owner } = createTwoPlugins()
    owner.plugin.register.tabKind({ kind: 'deck', component: STUB_COMPONENT })

    expect(() =>
      owner.plugin.register.tabKind({ kind: 'deck', component: STUB_COMPONENT }),
    ).toThrowError('Plugin "md.example.owner" registers the tab kind "deck" twice')
  })

  it('is free again once its owner disposes the registration', () => {
    const { host, owner, rival } = createTwoPlugins()
    owner.plugin.register.tabKind({ kind: 'deck', component: STUB_COMPONENT }).dispose()

    rival.plugin.register.tabKind({ kind: 'deck', title: 'Rival deck', component: STUB_COMPONENT })

    expect(host.registered.getRegisteredTabKinds().map((one) => one.title)).toStrictEqual([
      'Rival deck',
    ])
  })

  it('is free again once its owner is disposed', () => {
    const { host, owner, rival } = createTwoPlugins()
    owner.plugin.register.tabKind({ kind: 'deck', component: STUB_COMPONENT })

    host.lifecycle.disposePlugin('md.example.owner')

    expect(() =>
      rival.plugin.register.tabKind({ kind: 'deck', component: STUB_COMPONENT }),
    ).not.toThrow()
  })

  it('is read once from a registration whose kind changes between reads', () => {
    const { host, owner, rival } = createTwoPlugins()
    owner.plugin.register.tabKind({ kind: 'deck', title: 'Owner deck', component: STUB_COMPONENT })
    let reads = 0
    const shifting = {
      get kind() {
        reads += 1
        return reads === 1 ? 'reader:pdf' : 'deck'
      },
      title: 'Rival',
      component: STUB_COMPONENT,
    }

    rival.plugin.register.tabKind(shifting)

    expect(host.registered.getRegisteredTabKinds().map((one) => one.kind)).toStrictEqual([
      'deck',
      'reader:pdf',
    ])
  })
})

describe('a reader another plugin registers', () => {
  it('is refused, naming both plugins and the id, and the first plugin keeps it', () => {
    const { host, owner, rival } = createTwoPlugins()
    owner.plugin.register.reader({ id: 'pdf', title: 'Owner pdf', component: STUB_COMPONENT })

    expect(() =>
      rival.plugin.register.reader({ id: 'pdf', title: 'Rival pdf', component: STUB_COMPONENT }),
    ).toThrowError(
      'Plugin "md.example.rival" registers the reader "pdf", which plugin "md.example.owner" already registers',
    )

    expect(host.registered.getRegisteredReaders().map((one) => one.title)).toStrictEqual([
      'Owner pdf',
    ])
  })

  it('is refused when its prefixed id is the id of the reader of another plugin', () => {
    const { owner, rival } = createTwoPlugins()
    owner.plugin.register.reader({ id: 'pdf', component: STUB_COMPONENT })

    expect(() =>
      rival.plugin.register.reader({ id: 'reader:pdf', component: STUB_COMPONENT }),
    ).toThrowError('which plugin "md.example.owner" already registers')
  })

  it('is refused for the plugin that registers it twice', () => {
    const { owner } = createTwoPlugins()
    owner.plugin.register.reader({ id: 'pdf', component: STUB_COMPONENT })

    expect(() =>
      owner.plugin.register.reader({ id: 'pdf', component: STUB_COMPONENT }),
    ).toThrowError('Plugin "md.example.owner" registers the reader "pdf" twice')
  })

  it('is refused when its tab kind is a tab kind of another plugin', () => {
    const { owner, rival } = createTwoPlugins()
    owner.plugin.register.tabKind({ kind: 'reader:pdf', component: STUB_COMPONENT })

    expect(() =>
      rival.plugin.register.reader({ id: 'pdf', component: STUB_COMPONENT }),
    ).toThrowError(
      'Plugin "md.example.rival" registers the reader "pdf", which plugin "md.example.owner" already registers',
    )
  })

  it('refuses a tab kind that is the tab kind of the reader of another plugin', () => {
    const { owner, rival } = createTwoPlugins()
    owner.plugin.register.reader({ id: 'pdf', component: STUB_COMPONENT })

    expect(() =>
      rival.plugin.register.tabKind({ kind: 'reader:pdf', component: STUB_COMPONENT }),
    ).toThrowError(
      'Plugin "md.example.rival" registers the tab kind "reader:pdf", which plugin "md.example.owner" already registers',
    )
  })

  it('is refused for the plugin that registers a tab kind of the name the window gives it', () => {
    const { owner } = createTwoPlugins()
    owner.plugin.register.reader({ id: 'pdf', component: STUB_COMPONENT })

    expect(() =>
      owner.plugin.register.tabKind({ kind: 'reader:pdf', component: STUB_COMPONENT }),
    ).toThrowError('Plugin "md.example.owner" registers the tab kind "reader:pdf" twice')
  })

  it('is free again once its owner disposes the registration', () => {
    const { host, owner, rival } = createTwoPlugins()
    owner.plugin.register.reader({ id: 'pdf', component: STUB_COMPONENT }).dispose()

    rival.plugin.register.reader({ id: 'pdf', title: 'Rival pdf', component: STUB_COMPONENT })

    expect(host.registered.getRegisteredReaders().map((one) => one.title)).toStrictEqual([
      'Rival pdf',
    ])
  })

  it('is read once from a registration whose id changes between reads', () => {
    const { host, owner, rival } = createTwoPlugins()
    owner.plugin.register.reader({ id: 'pdf', title: 'Owner pdf', component: STUB_COMPONENT })
    let reads = 0
    const shifting = {
      get id() {
        reads += 1
        return reads === 1 ? 'reader:pdf' : 'pdf'
      },
      title: 'Rival',
      component: STUB_COMPONENT,
    }

    expect(() => rival.plugin.register.reader(shifting)).toThrowError('already registers')
    expect(host.registered.getRegisteredReaders().map((one) => one.title)).toStrictEqual([
      'Owner pdf',
    ])
  })
})
