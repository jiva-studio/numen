import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { FLOOR_SOURCE } from '../model/codes'
import { createFakeClient, createState } from '../model/fixtures'
import FloorPanel from './FloorPanel.vue'

globalThis.ResizeObserver ??= class {
  observe = vi.fn()
  unobserve = vi.fn()
  disconnect = vi.fn()
}

async function render(
  state = createState(),
  patch?: (client: ReturnType<typeof createFakeClient>) => void,
) {
  const client = createFakeClient(state)
  patch?.(client)
  const wrapper = mount(FloorPanel, {
    props: { pluginId: 'md.numen.engines.embeddings', values: {}, client },
  })
  await flushPromises()
  return { wrapper, client }
}

describe('FloorPanel', () => {
  it('shows a calibrated threshold with its noise and histogram', async () => {
    const { wrapper } = await render()
    expect(wrapper.get('[data-testid="floor-value"]').text()).toBe('0.41')
    expect(wrapper.get('[data-testid="floor-source"]').text()).toBe('Calibrated on your vault')
    expect(wrapper.get('[data-testid="floor-noise"]').text()).toBe('median 0.27 · spread 0.07')
    expect(wrapper.get('[data-testid="floor-note"]').text()).toContain(
      '200 titles × 500 passages · calibrated',
    )
    expect(wrapper.findAll('[data-testid="floor-bar"]')).toHaveLength(10)
    expect(wrapper.get('[data-testid="floor-histogram"]').attributes('aria-label')).toContain(
      'median 0.27',
    )
    expect(wrapper.find('[data-testid="floor-manual"]').exists()).toBe(false)
  })

  it('shows the model profile source with the pending note', async () => {
    const { wrapper } = await render(
      createState({ source: FLOOR_SOURCE.MODEL, calibration: undefined, floor: 0.3 }),
    )
    expect(wrapper.get('[data-testid="floor-source"]').text()).toBe('Model profile')
    expect(wrapper.get('[data-testid="floor-note"]').text()).toBe('Calibration pending')
    expect(wrapper.find('[data-testid="floor-histogram"]').exists()).toBe(false)
  })

  it('shows the not-enough message after a calibration with too few notes', async () => {
    const { wrapper, client } = await render(
      createState({ source: FLOOR_SOURCE.DEFAULT, calibration: undefined }),
      (fake) => fake.calibrateFloor.mockResolvedValue({ outcome: 2 }),
    )
    await wrapper.get('[data-testid="floor-recalculate"]').trigger('click')
    await flushPromises()
    expect(client.calibrateFloor).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="floor-note"]').text()).toBe(
      'Not enough notes yet (needs ≥ 20)',
    )
  })

  it('disables the recalculate button while calibrating', async () => {
    const releases: Array<() => void> = []
    const { wrapper } = await render(createState(), (fake) =>
      fake.calibrateFloor.mockImplementation(
        () => new Promise((resolve) => releases.push(() => resolve({ outcome: 1 }))),
      ),
    )
    await wrapper.get('[data-testid="floor-recalculate"]').trigger('click')
    const button = wrapper.get('[data-testid="floor-recalculate"]')
    expect(button.attributes('disabled')).toBeDefined()
    expect(button.text()).toBe('Calculating…')
    releases.forEach((release) => release())
    await flushPromises()
    expect(wrapper.get('[data-testid="floor-recalculate"]').attributes('disabled')).toBeUndefined()
  })

  it('shows the settings file note and disables the mode control', async () => {
    const { wrapper } = await render(createState({ source: FLOOR_SOURCE.FILE, floor: 0.5 }))
    expect(wrapper.get('[data-testid="floor-source"]').text()).toBe('Settings file')
    expect(wrapper.find('[data-testid="floor-file-note"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="floor-mode"]').attributes('data-disabled')).toBeDefined()
    expect(wrapper.find('[data-testid="floor-manual"]').exists()).toBe(false)
  })

  it('shows the labelled slider and the automatic marker in the manual mode', async () => {
    const { wrapper } = await render(
      createState({
        isManual: true,
        manualFloor: 0.6,
        floor: 0.6,
        source: FLOOR_SOURCE.MANUAL,
        automaticFloor: 0.41,
      }),
    )
    expect(wrapper.find('[data-testid="floor-manual"]').exists()).toBe(true)
    expect(wrapper.find('[aria-label="Manual threshold"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="floor-automatic-marker"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="floor-source"]').text()).toBe('Set by you')
  })

  it('switches to the manual mode through the segmented control', async () => {
    const { wrapper, client } = await render()
    const manual = wrapper.findAll('[role="radio"]').find((item) => item.text() === 'Manual')
    expect(manual).toBeDefined()
    await manual!.trigger('click')
    await flushPromises()
    expect(client.setFloor).toHaveBeenCalledWith({ isManual: true, floor: 0.41 })
    expect(wrapper.find('[data-testid="floor-manual"]').exists()).toBe(true)
  })

  it('shows the load error inline and loads again on retry', async () => {
    const client = createFakeClient()
    client.getFloor.mockRejectedValueOnce(new Error('[unavailable] embedder is not running'))
    const wrapper = mount(FloorPanel, { props: { pluginId: 'p', values: {}, client } })
    await flushPromises()
    expect(wrapper.get('[data-testid="floor-load-error"]').text()).toContain(
      'embedder is not running',
    )
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="floor-value"]').text()).toBe('0.41')
  })

  it('shows a calibration error inline', async () => {
    const { wrapper } = await render(createState(), (fake) =>
      fake.calibrateFloor.mockRejectedValue(new Error('[unavailable] no embedder')),
    )
    await wrapper.get('[data-testid="floor-recalculate"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-testid="floor-action-error"]').text()).toContain('no embedder')
  })

  it('reports a probe that passes', async () => {
    const { wrapper, client } = await render()
    await wrapper.get('[data-testid="floor-probe-input"]').setValue('love')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(client.probeFloor).toHaveBeenCalledWith({ query: 'love' })
    expect(wrapper.get('[data-testid="floor-probe-result"]').text()).toBe(
      'Best match 0.46 — above the threshold (passes)',
    )
  })

  it('reports a probe that falls below the threshold', async () => {
    const { wrapper } = await render(createState(), (fake) =>
      fake.probeFloor.mockResolvedValue({ bestSimilarity: 0.3, floor: 0.41, doesPass: false }),
    )
    await wrapper.get('[data-testid="floor-probe-input"]').setValue('zzz')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.get('[data-testid="floor-probe-result"]').text()).toBe(
      'Best match 0.30 — below the threshold',
    )
  })

  it('shows a probe error inline and does not check an empty query', async () => {
    const { wrapper, client } = await render(createState(), (fake) =>
      fake.probeFloor.mockRejectedValue(new Error('[unavailable] no embedder')),
    )
    expect(wrapper.get('[data-testid="floor-probe-button"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[data-testid="floor-probe-input"]').setValue('x')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(client.probeFloor).toHaveBeenCalledTimes(1)
    expect(wrapper.get('[data-testid="floor-probe-error"]').text()).toContain('no embedder')
    expect(wrapper.find('[data-testid="floor-probe-result"]').exists()).toBe(false)
  })

  it('reloads when the model setting changes', async () => {
    const { wrapper, client } = await render()
    await wrapper.setProps({ values: { model: 'Xenova/multilingual-e5-small' } })
    await flushPromises()
    expect(client.getFloor).toHaveBeenCalledTimes(2)
  })
})
