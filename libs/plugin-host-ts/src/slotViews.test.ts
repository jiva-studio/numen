import { describe, expect, it, vi } from 'vitest'
import { createSlotViewRegistry, getDeclaredSlot } from './slotViews'

const BAR = { id: 'bar', component: { name: 'OrchardBar' } }
const OTHER_BAR = { id: 'strip', component: { name: 'HarborBar' } }

describe('createSlotViewRegistry', () => {
  it('draws no view in a slot no plugin registered', () => {
    const registry = createSlotViewRegistry()

    expect(registry.getSlotView('titleBar').value).toBeNull()
  })

  it('draws the view a plugin registered for the slot', () => {
    const registry = createSlotViewRegistry()
    const view = registry.getSlotView('titleBar')

    registry.registerSlotView('org.example.orchard', 'titleBar', BAR)

    expect(view.value).toMatchObject({ pluginId: 'org.example.orchard', registration: BAR })
  })

  it('draws the view of the plugin first by id when two plugins register the slot', () => {
    const registry = createSlotViewRegistry()

    registry.registerSlotView('org.example.orchard', 'titleBar', BAR)
    registry.registerSlotView('org.example.harbor', 'titleBar', OTHER_BAR)

    expect(registry.getSlotView('titleBar').value?.pluginId).toBe('org.example.harbor')
  })

  it('reports the plugin registered second whose id sorts after the first', () => {
    const reportIssue = vi.fn()
    const registry = createSlotViewRegistry({ reportIssue })

    registry.registerSlotView('org.example.harbor', 'titleBar', OTHER_BAR)
    registry.registerSlotView('org.example.orchard', 'titleBar', BAR)

    expect(reportIssue).toHaveBeenCalledOnce()
    expect(reportIssue).toHaveBeenCalledWith(
      'org.example.orchard',
      expect.stringContaining('org.example.harbor'),
    )
  })

  it('reports the plugin that drew the slot when a plugin first by id registers it', () => {
    const reportIssue = vi.fn()
    const registry = createSlotViewRegistry({ reportIssue })

    registry.registerSlotView('org.example.orchard', 'titleBar', BAR)
    registry.registerSlotView('org.example.harbor', 'titleBar', OTHER_BAR)

    expect(reportIssue).toHaveBeenCalledOnce()
    expect(reportIssue.mock.calls[0]?.[0]).toBe('org.example.orchard')
  })

  it('reports nothing for one plugin that registers its view again', () => {
    const reportIssue = vi.fn()
    const registry = createSlotViewRegistry({ reportIssue })

    registry.registerSlotView('org.example.orchard', 'titleBar', BAR)
    registry.registerSlotView('org.example.orchard', 'titleBar', BAR)

    expect(reportIssue).not.toHaveBeenCalled()
    expect(registry.views.value).toHaveLength(1)
  })

  it('draws the other plugin once the plugin that drew the slot is disposed', () => {
    const registry = createSlotViewRegistry()
    registry.registerSlotView('org.example.harbor', 'titleBar', OTHER_BAR)
    registry.registerSlotView('org.example.orchard', 'titleBar', BAR)

    registry.disposePlugin('org.example.harbor')

    expect(registry.getSlotView('titleBar').value?.pluginId).toBe('org.example.orchard')
  })

  it('draws nothing once the registration is disposed', () => {
    const registry = createSlotViewRegistry()
    const registration = registry.registerSlotView('org.example.orchard', 'titleBar', BAR)

    registration.dispose()

    expect(registry.getSlotView('titleBar').value).toBeNull()
  })
})

describe('getDeclaredSlot', () => {
  const manifest = { contributes: { views: [{ id: 'bar', slot: 'titleBar' }] } }

  it('returns the slot the manifest declares for the id', () => {
    expect(getDeclaredSlot('org.example.orchard', manifest, 'bar')).toBe('titleBar')
  })

  it('throws for an id the manifest does not declare', () => {
    expect(() => getDeclaredSlot('org.example.orchard', manifest, 'strip')).toThrow(
      /contributes\.views does not declare/,
    )
  })

  it('throws where the plugin has no manifest', () => {
    expect(() => getDeclaredSlot('org.example.orchard', undefined, 'bar')).toThrow()
  })
})
