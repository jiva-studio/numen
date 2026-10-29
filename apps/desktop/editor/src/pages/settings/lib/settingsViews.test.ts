import { describe, expect, it } from 'vitest'
import { getRegisteredViewsAfter, getSettingValues } from './settingsViews'
import { createPluginInfo } from '@/testing/installation'

describe('getRegisteredViewsAfter', () => {
  const declared = [
    { id: 'top', title: 'Top', after: '' },
    { id: 'floor', title: 'Floor', after: 'model' },
    { id: 'missing', title: 'Missing', after: 'model' },
  ]
  const registered = new Map([
    ['top', { component: 'T' }],
    ['floor', { component: 'F' }],
  ])

  it('returns the registered views placed after a key, in declared order', () => {
    expect(getRegisteredViewsAfter(declared, registered, 'model')).toStrictEqual([
      { id: 'floor', component: 'F' },
    ])
  })

  it('places a view with an empty after at the empty key', () => {
    expect(getRegisteredViewsAfter(declared, registered, '')).toStrictEqual([
      { id: 'top', component: 'T' },
    ])
  })

  it('returns nothing while no view is registered', () => {
    expect(getRegisteredViewsAfter(declared, new Map(), 'model')).toStrictEqual([])
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
