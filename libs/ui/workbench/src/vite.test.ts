// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { getUserAgent } from './shared/api/dom'
import { LIGHTNING_CSS, createWindowAliases, createWindowConfig } from './vite'

const ROOT = 'file:///repo/apps/desktop/sample/vite.config.ts'

describe('createWindowAliases', () => {
  it('resolves the window source and the shared packages against the config file', () => {
    const aliases = createWindowAliases(ROOT)

    expect(aliases['@']).toBe('/repo/apps/desktop/sample/src')
    expect(aliases['@numen/ui']).toBe('/repo/libs/ui/components/dist/index.js')
    expect(aliases['@numen/core-client']).toBe('/repo/libs/core/client/src/index.ts')
  })

  it('lists the aliases of the window before the package a subpath of theirs starts with', () => {
    const aliases = createWindowAliases(ROOT, {
      '@numen/core-client/testing': '../../../libs/core/client/src/testing/index.ts',
    })

    const names = Object.keys(aliases)
    expect(names.indexOf('@numen/core-client/testing')).toBeLessThan(
      names.indexOf('@numen/core-client'),
    )
    expect(aliases['@numen/core-client/testing']).toBe(
      '/repo/libs/core/client/src/testing/index.ts',
    )
  })
})

describe('createWindowConfig', () => {
  const hostModules = {
    createHostModuleEntries: () => ({ name: 'host-module-entries' }),
    hostModuleInputs: { 'sdk/vue': 'host-module:vue' },
    getEntryFileName: () => '[name].js',
  }

  it('builds the page and the host modules into the Go package named', () => {
    const config = createWindowConfig(ROOT, {
      plugins: [{ name: 'vue' }],
      hostModules,
      outDir: '../../../libs/core/kernel/adapter/window/sample/pages/app',
    })

    expect(config.plugins).toStrictEqual([{ name: 'vue' }, { name: 'host-module-entries' }])
    expect(config.css).toStrictEqual({ lightningcss: LIGHTNING_CSS })
    expect(config.resolve).toStrictEqual({ alias: createWindowAliases(ROOT) })
    expect(config.build?.outDir).toBe('../../../libs/core/kernel/adapter/window/sample/pages/app')
    expect(config.build?.emptyOutDir).toBe(true)
    expect(config.build?.assetsDir).toBe('built')
    expect(config.build?.rollupOptions?.preserveEntrySignatures).toBe('strict')
    expect(config.build?.rollupOptions?.input).toStrictEqual({
      index: '/repo/apps/desktop/sample/index.html',
      'sdk/vue': 'host-module:vue',
    })
    expect(config.build?.rollupOptions?.output).toStrictEqual({
      entryFileNames: hostModules.getEntryFileName,
    })
    expect(getUserAgent()).toBe(globalThis.navigator.userAgent)
  })
})
