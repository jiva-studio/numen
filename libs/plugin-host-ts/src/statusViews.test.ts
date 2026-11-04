import { describe, expect, it } from 'vitest'
import { createStatusViewRegistry } from './statusViews'

const view = (id: string) => ({ id, component: {} })

describe('the status view registry', () => {
  it('lists the views in the order they were registered', () => {
    const registry = createStatusViewRegistry()

    registry.registerStatusView('a.one', view('first'))
    registry.registerStatusView('b.two', view('second'))

    expect(registry.views.value.map((one) => [one.pluginId, one.registration.id])).toEqual([
      ['a.one', 'first'],
      ['b.two', 'second'],
    ])
  })

  it('removes a view when its registration is disposed', () => {
    const registry = createStatusViewRegistry()
    const first = registry.registerStatusView('a.one', view('first'))
    registry.registerStatusView('a.one', view('second'))

    first.dispose()

    expect(registry.views.value.map((one) => one.registration.id)).toEqual(['second'])
  })

  it('replaces the view a plugin registers under an id again', () => {
    const registry = createStatusViewRegistry()
    const earlier = registry.registerStatusView('a.one', view('first'))
    const other = view('second')
    registry.registerStatusView('a.one', other)
    const later = { id: 'first', component: { replaced: true } }

    registry.registerStatusView('a.one', later)
    expect(registry.views.value.map((one) => one.registration)).toEqual([other, later])

    earlier.dispose()
    expect(registry.views.value.map((one) => one.registration)).toEqual([other, later])
  })

  it('keeps the views of two plugins that use the same id apart', () => {
    const registry = createStatusViewRegistry()
    registry.registerStatusView('a.one', view('update'))
    registry.registerStatusView('b.two', view('update'))

    registry.disposePlugin('a.one')

    expect(registry.views.value.map((one) => one.pluginId)).toEqual(['b.two'])
  })
})
