import { describe, expect, it } from 'vitest'
import { getRegisteredViews, getSettingValues } from './settingsViews'
import { createPluginInfo } from '@/testing/installation'

describe('getRegisteredViews', () => {
  const declared = [
    { id: 'top', title: 'Top' },
    { id: 'floor', title: 'Floor' },
    { id: 'missing', title: 'Missing' },
  ]
  const registered = new Map([
    ['top', { component: 'T' }],
    ['floor', { component: 'F' }],
  ])

  it('returns the registered views in declared order', () => {
    expect(getRegisteredViews(declared, registered)).toStrictEqual([
      { id: 'top', component: 'T' },
      { id: 'floor', component: 'F' },
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
    expect(getSettingValues(plugin, (key) => (key === 'a' ? 5 : undefined))).toStrictEqual({
      a: 5,
      b: '',
    })
  })
})
