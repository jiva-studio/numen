// @vitest-environment node
/** The two limits of a plugin start: the wait of the window, and the start limit of a frontend, which is the one of the backend activation. */
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { INTERFACE_WAIT_MS, PLUGIN_START_LIMIT_MS } from './limits'

const ACTIVATION = fileURLToPath(
  new URL('../../../../core/kernel/internal/adapter/plugin/lifecycle/activation.go', import.meta.url),
)

describe('the wait of a window for the plugins of its interface', () => {
  it('is 3 s', () => {
    expect(INTERFACE_WAIT_MS).toBe(3_000)
  })
})

describe('the start limit of a plugin', () => {
  it('is 30 s, as the backend activation takes', () => {
    expect(PLUGIN_START_LIMIT_MS).toBe(30_000)
    expect(readFileSync(ACTIVATION, 'utf8')).toMatch(/Timeout:\s*30 \* time\.Second/)
  })
})
