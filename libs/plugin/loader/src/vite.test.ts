import { describe, expect, it } from 'vitest'
import { HOST_MODULES } from '@numen/plugin-sdk/host-modules'
import { createHostModuleEntries, getEntryFileName, hostModuleInputs } from './vite'

describe('vite host module entries', () => {
  it('resolves and loads virtual host-module entries', () => {
    const plugin = createHostModuleEntries()
    expect(plugin.name).toBe('host-module-entries')

    const resolveId = plugin.resolveId as (id: string) => string | undefined
    const load = plugin.load as (id: string) => string | undefined

    expect(resolveId('host-module:vue')).toBe('\0host-module:vue')
    expect(resolveId('other/host-module:vue')).toBeUndefined()
    expect(resolveId('vue')).toBeUndefined()

    expect(load('\0host-module:vue')).toBe("export * from 'vue'")
    expect(load('prefix\0host-module:vue')).toBeUndefined()
    expect(load('host-module:vue')).toBeUndefined()
  })

  it('maps every HOST_MODULES entry to its stripped output name', () => {
    expect(Object.keys(hostModuleInputs).length).toBe(Object.keys(HOST_MODULES).length)
    expect(hostModuleInputs['sdk/vue']).toBe('host-module:vue')
    for (const [name, path] of Object.entries(HOST_MODULES)) {
      const key = path.slice(1, -3)
      expect(hostModuleInputs[key]).toBe(`host-module:${name}`)
    }
  })

  it('serves one CodeMirror and one Lezer to every plugin', () => {
    expect(hostModuleInputs['sdk/codemirror/state']).toBe('host-module:@codemirror/state')
    expect(hostModuleInputs['sdk/codemirror/view']).toBe('host-module:@codemirror/view')
    expect(hostModuleInputs['sdk/codemirror/language']).toBe('host-module:@codemirror/language')
    expect(hostModuleInputs['sdk/lezer/common']).toBe('host-module:@lezer/common')
    expect(hostModuleInputs['sdk/lezer/markdown']).toBe('host-module:@lezer/markdown')
  })

  it('loads a scoped host module by its full name', () => {
    const load = createHostModuleEntries().load as (id: string) => string | undefined
    expect(load('\0host-module:@codemirror/state')).toBe("export * from '@codemirror/state'")
  })

  it('writes host modules to unhashed paths and other chunks to built/', () => {
    expect(getEntryFileName({ name: 'sdk/vue' })).toBe('[name].js')
    expect(getEntryFileName({ name: 'sdk/codemirror/state' })).toBe('[name].js')
    expect(getEntryFileName({ name: 'index' })).toBe('built/[name]-[hash].js')
  })
})
