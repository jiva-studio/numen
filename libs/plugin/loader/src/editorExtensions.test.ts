import { describe, expect, it, vi } from 'vitest'
import type { EditorExtensionEntry } from '@numen/plugin-sdk'
import { createEditorExtensionRegistry } from './editorExtensions'

const extension = [] as const

describe('createEditorExtensionRegistry', () => {
  it('lists the registered extensions in the order they were registered', () => {
    const registry = createEditorExtensionRegistry()
    registry.registerEditorExtension('md.a', { id: 'one', extension })
    registry.registerEditorExtension('md.b', { id: 'two', extension })

    expect(registry.get().map((one) => `${one.pluginId}/${one.id}`)).toEqual([
      'md.a/one',
      'md.b/two',
    ])
  })

  it('tells the listeners when an extension is registered and disposed', () => {
    const registry = createEditorExtensionRegistry()
    const seen: number[] = []
    registry.onChange((entries) => seen.push(entries.length))

    const registration = registry.registerEditorExtension('md.a', { id: 'one', extension })
    registration.dispose()
    registration.dispose()

    expect(seen).toEqual([1, 0])
    expect(registry.get()).toEqual([])
  })

  it('stops calling a listener that was disposed', () => {
    const registry = createEditorExtensionRegistry()
    const handler = vi.fn()
    registry.onChange(handler).dispose()

    registry.registerEditorExtension('md.a', { id: 'one', extension })

    expect(handler).not.toHaveBeenCalled()
  })

  it('refuses the same id twice from one plugin, and takes it again after dispose', () => {
    const registry = createEditorExtensionRegistry()
    const first = registry.registerEditorExtension('md.a', { id: 'one', extension })

    expect(() => registry.registerEditorExtension('md.a', { id: 'one', extension })).toThrowError(
      'Plugin "md.a" registers the editor extension "one" twice',
    )
    first.dispose()
    expect(() => registry.registerEditorExtension('md.a', { id: 'one', extension })).not.toThrow()
  })

  it('leaves out a disabled extension and brings it back when enabled', () => {
    const registry = createEditorExtensionRegistry()
    registry.registerEditorExtension('md.a', { id: 'one', extension })

    registry.setEnabled('md.a', 'one', false)
    expect(registry.get()).toEqual([])

    registry.setEnabled('md.a', 'one', true)
    expect(registry.get()).toHaveLength(1)
  })

  it('keeps an extension off that was disabled before it registered', () => {
    const registry = createEditorExtensionRegistry()
    registry.setEnabled('md.a', 'one', false)

    registry.registerEditorExtension('md.a', { id: 'one', extension })

    expect(registry.get()).toEqual([])
  })

  it('switches off an extension that failed, reports it under its plugin and keeps the others', () => {
    const reportError = vi.fn()
    const registry = createEditorExtensionRegistry({ reportError })
    registry.registerEditorExtension('md.a', { id: 'one', extension })
    registry.registerEditorExtension('md.b', { id: 'two', extension })
    const failing = registry.get()[0] as EditorExtensionEntry
    const error = new Error('broken')

    registry.fail(failing, error)
    registry.fail(failing, error)

    expect(registry.get().map((one) => one.id)).toEqual(['two'])
    expect(reportError).toHaveBeenCalledTimes(1)
    expect(reportError).toHaveBeenCalledWith('md.a', error)
  })

  it('keeps a failed extension off when its plugin registers it again', () => {
    const registry = createEditorExtensionRegistry()
    const registration = registry.registerEditorExtension('md.a', { id: 'one', extension })
    registry.fail(registry.get()[0] as EditorExtensionEntry, new Error('broken'))
    registration.dispose()

    registry.registerEditorExtension('md.a', { id: 'one', extension })

    expect(registry.get()).toEqual([])
  })
})
