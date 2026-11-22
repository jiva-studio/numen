import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { EditorView } from '@codemirror/view'
import { StateField, type Extension } from '@codemirror/state'
import type { EditorExtensionEntry, EditorExtensionHost, HostServices } from '@numen/plugin-sdk'
import MarkdownNoteEditor from './MarkdownNoteEditor.vue'
import { activate } from '../index'
import { createEditorSettingsContext } from '../testing/editorSettings'
import { getPluginExtensions, pluginExtensionEntries } from '../model/pluginExtensions'

const NOTE = 'first line\nsecond line\n\nelsewhere'

function createHost(initial: readonly EditorExtensionEntry[] = []) {
  let entries = initial
  const handlers = new Set<(entries: readonly EditorExtensionEntry[]) => void>()
  const fail = vi.fn()
  const editorExtensions: EditorExtensionHost = {
    get: () => entries,
    onChange: (handler) => {
      handlers.add(handler)
      return { dispose: () => handlers.delete(handler) }
    },
    fail,
  }
  activate(createEditorSettingsContext({ lineNumbers: false, softWrap: true }).context, {
    vault: {},
    editor: editorExtensions,
  } as unknown as HostServices)
  return {
    fail,
    set(next: readonly EditorExtensionEntry[]) {
      entries = next
      for (const handler of handlers) handler(entries)
    },
  }
}

const entry = (id: string, extension: Extension): EditorExtensionEntry => ({
  pluginId: 'md.example',
  id,
  extension,
})

const mountEditor = () =>
  mount(MarkdownNoteEditor, {
    props: { modelValue: NOTE, filePath: 'a.md' },
    attachTo: document.body,
  })

afterEach(() => {
  pluginExtensionEntries.value = []
  document.body.innerHTML = ''
})

describe('the extensions other plugins add to the editor', () => {
  it('draws no widget while no plugin adds an extension', async () => {
    createHost()
    const wrapper = mountEditor()
    await flushPromises()

    expect(wrapper.find('[data-added]').exists()).toBe(false)
    expect(wrapper.text()).toContain('first line')
  })

  it('applies a registered extension and drops it when the plugin disposes it', async () => {
    const marker = EditorView.editorAttributes.of({ 'data-added': 'yes' })
    const host = createHost()
    const wrapper = mountEditor()
    await flushPromises()
    expect(wrapper.find('[data-added]').exists()).toBe(false)

    host.set([entry('marker', marker)])
    await nextTick()
    await flushPromises()
    expect(wrapper.find('[data-added="yes"]').exists()).toBe(true)

    host.set([])
    await nextTick()
    await flushPromises()
    expect(wrapper.find('[data-added]').exists()).toBe(false)
  })

  it('switches off an extension that cannot be built and keeps the editor working', async () => {
    const broken = StateField.define({
      create() {
        throw new Error('broken')
      },
      update: (value) => value,
    })
    const host = createHost([entry('broken', broken)])
    const wrapper = mountEditor()
    await flushPromises()

    expect(host.fail).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('first line')
  })

  it('reports a failing extension after the read that found it, not during it', async () => {
    const broken = StateField.define({
      create() {
        throw new Error('broken')
      },
      update: (value) => value,
    })
    const host = createHost()

    getPluginExtensions([entry('broken', broken)])
    expect(host.fail).not.toHaveBeenCalled()

    await Promise.resolve()
    expect(host.fail).toHaveBeenCalledTimes(1)
  })
})
