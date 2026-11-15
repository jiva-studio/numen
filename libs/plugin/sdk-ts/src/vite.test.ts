import { describe, expect, it } from 'vitest'
import { createPluginBuildConfig } from './vite.ts'

function getRollupOptions() {
  const config = createPluginBuildConfig({ root: '/plugin', plugins: [] })
  return config.build?.rollupOptions as {
    external: readonly string[]
    output: { paths: Readonly<Record<string, string>> }
  }
}

describe('createPluginBuildConfig', () => {
  it('imports CodeMirror and Lezer from the window instead of bundling them', () => {
    const { external, output } = getRollupOptions()
    for (const name of ['@codemirror/state', '@codemirror/view', '@lezer/common']) {
      expect(external).toContain(name)
    }
    expect(output.paths['@codemirror/state']).toBe('/sdk/codemirror/state.js')
    expect(output.paths['@lezer/common']).toBe('/sdk/lezer/common.js')
  })

  it('imports Vue from the window', () => {
    const { external, output } = getRollupOptions()
    expect(external).toContain('vue')
    expect(output.paths.vue).toBe('/sdk/vue.js')
  })
})
