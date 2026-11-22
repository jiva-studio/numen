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

  it('notifies the listeners on registering and disposing an extension', () => {
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

  it('ignores a failure of an entry that is not registered', () => {
    const reportError = vi.fn()
    const registry = createEditorExtensionRegistry({ reportError })
    registry.registerEditorExtension('md.a', { id: 'one', extension })

    registry.fail({ pluginId: 'md.a', id: 'one', extension }, new Error('forged'))

    expect(registry.get()).toHaveLength(1)
    expect(reportError).not.toHaveBeenCalled()
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
