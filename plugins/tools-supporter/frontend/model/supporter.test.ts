import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { createSupporterState, SUPPORTER_LICENSE_ID } from './supporter'

describe('createSupporterState', () => {
  it('starts with the initial supporter license state and updates on change', () => {
    const context = createMockPluginContext()
    const state = createSupporterState(context.licenses)
    state.start()

    expect(state.isActive.value).toBe(false)

    context.setLicense(SUPPORTER_LICENSE_ID, true)
    expect(state.isActive.value).toBe(true)

    context.setLicense(SUPPORTER_LICENSE_ID, false)
    expect(state.isActive.value).toBe(false)
  })

  it('reads an active license on start and stops listening on stop', () => {
    const context = createMockPluginContext({
      initialLicenses: { [SUPPORTER_LICENSE_ID]: true },
    })
    const state = createSupporterState(context.licenses)
    state.start()
    expect(state.isActive.value).toBe(true)

    state.stop()
    context.setLicense(SUPPORTER_LICENSE_ID, false)
    expect(state.isActive.value).toBe(true)
  })
})
