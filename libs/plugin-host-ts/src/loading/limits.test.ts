// @vitest-environment node
/** The start limit of a frontend, which is the one of the backend activation. */
import { readFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { PLUGIN_START_LIMIT_MS } from './limits'

const ACTIVATION = fileURLToPath(
  new URL('../../../core/internal/adapter/plugin/lifecycle/activation.go', import.meta.url),
)

describe('the start limit of a plugin', () => {
  it('is 30 s, as the backend activation takes', () => {
    expect(PLUGIN_START_LIMIT_MS).toBe(30_000)
    expect(readFileSync(ACTIVATION, 'utf8')).toMatch(/Timeout:\s*30 \* time\.Second/)
  })
})
