import { describe, expect, it } from 'vitest'
import { getRegisteredViews } from './settingsViews'

describe('getRegisteredViews of the page of a plugin', () => {
  const declared = [
    { id: 'vaults', title: 'Vaults', page: '' },
    { id: 'subscription', title: 'Subscription', page: 'identity' },
    { id: 'floor', title: 'Floor', page: '' },
  ]
  const VAULTS = { name: 'Vaults' }
  const SUBSCRIPTION = { name: 'Subscription' }
  const FLOOR = { name: 'Floor' }
  const registered = new Map([
    ['vaults', { component: VAULTS }],
    ['subscription', { component: SUBSCRIPTION }],
    ['floor', { component: FLOOR }],
  ])

  it('leaves out a view that belongs to the page of the identity', () => {
    expect(getRegisteredViews(declared, registered).map((one) => one.id)).toStrictEqual([
      'vaults',
      'floor',
    ])
  })

  it('keeps a view whose page is empty', () => {
    expect(getRegisteredViews(declared.slice(0, 1), registered)).toStrictEqual([
      { id: 'vaults', component: VAULTS },
    ])
  })
})
