import { describe, expect, it } from 'vitest'
import type { PluginHealthItem } from '@/entities/plugin'
import { getManagedHealth, getMarkedSettings } from './pluginHealth'
import { getRegisteredViews, getSettingValues } from './settingsViews'
import { hasPluginSettings } from './pluginSettings'
import { WORDS as words } from '../words'
import { createPluginInfo } from '@/pages/settings/testing/installation'

describe('getRegisteredViews', () => {
  const declared = [
    { id: 'top', title: 'Top' },
    { id: 'floor', title: 'Floor' },
    { id: 'missing', title: 'Missing' },
  ]
  const T = { name: 'T' }
  const F = { name: 'F' }
  const registered = new Map([
    ['top', { component: T }],
    ['floor', { component: F }],
  ])

  it('returns the registered views in declared order', () => {
    expect(getRegisteredViews(declared, registered)).toStrictEqual([
      { id: 'top', component: T },
      { id: 'floor', component: F },
    ])
  })

  it('returns nothing while no view is registered', () => {
    expect(getRegisteredViews(declared, new Map())).toStrictEqual([])
  })
})

describe('getSettingValues', () => {
  it('reads every declared setting by key', () => {
    const plugin = createPluginInfo('p', 'P', {
      settings: [
        { key: 'a', type: 'number', title: 'A', description: '', default: 1, palette: false },
        { key: 'b', type: 'string', title: 'B', description: '', default: '', palette: false },
      ],
    })
    expect(
      getSettingValues(
        plugin,
        (key) => (key === 'a' ? 5 : undefined),
        () => false,
      ),
    ).toStrictEqual({
      a: 5,
      b: '',
    })
  })
})

describe('hasPluginSettings', () => {
  it('detects non-connection settings and settingsViews', () => {
    const empty = createPluginInfo('p', 'P')
    expect(hasPluginSettings(empty)).toBe(false)

    const connectionsOnly = createPluginInfo('p', 'P', {
      settings: [
        { key: 'c', type: 'connections', title: 'C', description: '', default: undefined },
      ],
    })
    expect(hasPluginSettings(connectionsOnly)).toBe(false)

    const withSetting = createPluginInfo('p', 'P', {
      settings: [{ key: 'a', type: 'string', title: 'A', description: '', default: '' }],
    })
    expect(hasPluginSettings(withSetting)).toBe(true)

    const withView = createPluginInfo('p', 'P', {
      settingsViews: [{ id: 'v', title: 'V' }],
    })
    expect(hasPluginSettings(withView)).toBe(true)
  })
})

describe('getManagedHealth and getMarkedSettings', () => {
  const items: readonly PluginHealthItem[] = [
    {
      pluginId: 'p1',
      key: 'titled',
      severity: 'error',
      text: 'Has custom title',
      detail: 'd1',
      action: { command: 'cmd.allowed', values: { setting: 'source' }, title: 'Open Source' },
    },
    {
      pluginId: 'p1',
      key: 'untitled',
      severity: 'warning',
      text: 'Uses fallback title',
      detail: 'd2',
      action: { command: 'cmd.allowed', values: { setting: '' }, title: '' },
    },
    {
      pluginId: 'p1',
      key: 'disallowed',
      severity: 'warning',
      text: 'Cannot run command',
      detail: 'd3',
      action: { command: 'cmd.denied', values: {}, title: 'Ignored' },
    },
    {
      pluginId: 'p1',
      key: 'no-action',
      severity: 'error',
      text: 'No action',
      detail: 'd4',
    },
    {
      pluginId: 'p2',
      key: 'other',
      severity: 'error',
      text: 'Other plugin',
      detail: 'd5',
      action: { command: 'cmd.allowed', values: { setting: 'otherSetting' }, title: 'Other' },
    },
  ]

  it('filters health items by plugin and resolves action labels', () => {
    expect(getManagedHealth(items, 'p1', (cmd) => cmd === 'cmd.allowed')).toStrictEqual([
      {
        key: 'titled',
        severity: 'error',
        text: 'Has custom title',
        detail: 'd1',
        actionLabel: 'Open Source',
      },
      {
        key: 'untitled',
        severity: 'warning',
        text: 'Uses fallback title',
        detail: 'd2',
        actionLabel: words.healthAction,
      },
      {
        key: 'disallowed',
        severity: 'warning',
        text: 'Cannot run command',
        detail: 'd3',
        actionLabel: '',
      },
      {
        key: 'no-action',
        severity: 'error',
        text: 'No action',
        detail: 'd4',
        actionLabel: '',
      },
    ])
  })

  it('returns only non-empty setting keys for the requested plugin', () => {
    expect(getMarkedSettings(items, 'p1')).toStrictEqual(['source'])
    expect(getMarkedSettings(items, 'p2')).toStrictEqual(['otherSetting'])
    expect(getMarkedSettings(items, 'p3')).toStrictEqual([])
  })
})

describe('settings words helpers', () => {
  it('formats search, placeholder, and singular/plural index questions', () => {
    expect(words.searchValues('Model')).toBe('Search model')
    expect(words.connectionRowsValuePlaceholder('Model')).toBe('Choose or type a model')
    expect(words.connectionRowsIndexQuestion(1, 'Home')).toBe(
      'The vault will be indexed again: 1 note through Home. This can take a while and may cost money.',
    )
    expect(words.connectionRowsIndexQuestion(2, 'Home')).toBe(
      'The vault will be indexed again: 2 notes through Home. This can take a while and may cost money.',
    )
  })
})
