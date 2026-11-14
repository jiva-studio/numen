/**
 * A setting of type list on the page of a plugin: the items in force, and each add, remove and reset
 * written as the whole list. The installation holds a write as requested until the vault answers, as
 * the window does, so a second add is built on the first. Every value here is invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { mount } from '@vue/test-utils'
import PluginDetail from './PluginDetail.vue'
import { ListField } from '../list-field'
import { createInstallation, createPluginInfo } from '@/pages/settings/testing/installation'
import { getSettingAt as at } from '@/pages/settings/testing/settingAt'
import type { PluginInfo, PluginSettingInfo, SettingEdit } from '@/entities/settings'
import type { JsonObject, JsonValue } from '@/shared/api/json'

beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe = vi.fn()
      unobserve = vi.fn()
      disconnect = vi.fn()
    },
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

const PLUGIN_ID = 'org.example.tasks'

const FOLDERS: PluginSettingInfo = {
  key: 'ignoredFolders',
  type: 'list',
  title: 'Ignored folders',
  description: 'Folders whose notes are not searched for tasks',
  default: [],
}

const TASKS: PluginInfo = createPluginInfo(PLUGIN_ID, 'Tasks', { settings: [FOLDERS] })

const createListDetail = (file: JsonObject = {}, plugin: PluginInfo = TASKS) => {
  const writes: SettingEdit[] = []
  const requested = new Map<string, JsonValue>()
  const installation = createInstallation({
    getSetting: (path) =>
      requested.has(path.join('/')) ? requested.get(path.join('/')) : at(file, path),
    isRequested: (path) => requested.has(path.join('/')),
    write: (edits) => {
      writes.push(...edits)
      for (const edit of edits)
        requested.set(edit.at.join('/'), JSON.parse(edit.value) as JsonValue)
    },
    plugins: ref([plugin]),
  })
  const wrapper = mount(PluginDetail, {
    props: {
      settings: installation.settings,
      manager: installation.plugin,
      connections: installation.connections,
      plugin,
    },
  })
  const getField = () => wrapper.getComponent(ListField)
  const readWrites = () =>
    writes.map((one) => ({ at: one.at, value: JSON.parse(one.value) as JsonValue }))
  return { wrapper, getField, readWrites }
}

const findResets = (wrapper: ReturnType<typeof createListDetail>['wrapper']) =>
  wrapper.findAll('button').filter((one) => one.attributes('aria-label')?.startsWith('Reset'))

describe('a list setting on the page of a plugin', () => {
  it('draws the list control with the items stored', () => {
    const { getField } = createListDetail({
      [PLUGIN_ID]: { ignoredFolders: ['templates', 'archive/old'] },
    })

    expect(getField().props('items')).toStrictEqual(['templates', 'archive/old'])
  })

  it('draws the default while no list is stored', () => {
    const plugin = createPluginInfo(PLUGIN_ID, 'Tasks', {
      settings: [{ ...FOLDERS, default: ['templates'] }],
    })
    const { getField } = createListDetail({}, plugin)

    expect(getField().props('items')).toStrictEqual(['templates'])
  })

  it('draws only the strings of a stored list', () => {
    const { getField } = createListDetail({
      [PLUGIN_ID]: { ignoredFolders: ['templates', 3, null] },
    })

    expect(getField().props('items')).toStrictEqual(['templates'])
  })

  it('draws an empty list for a stored value that is no list', () => {
    const { getField } = createListDetail({ [PLUGIN_ID]: { ignoredFolders: 'templates,archive' } })

    expect(getField().props('items')).toStrictEqual([])
  })

  it('writes an added item as the last item of the list', () => {
    const { getField, readWrites } = createListDetail({
      [PLUGIN_ID]: { ignoredFolders: ['templates', 'archive/old'] },
    })

    getField().vm.$emit('add', 'drafts')

    expect(readWrites()).toStrictEqual([
      { at: [PLUGIN_ID, 'ignoredFolders'], value: ['templates', 'archive/old', 'drafts'] },
    ])
  })

  it('writes the list without the item removed and keeps the others', () => {
    const { getField, readWrites } = createListDetail({
      [PLUGIN_ID]: { ignoredFolders: ['templates', 'archive/old', 'drafts'] },
    })

    getField().vm.$emit('remove', 'archive/old')

    expect(readWrites()).toStrictEqual([
      { at: [PLUGIN_ID, 'ignoredFolders'], value: ['templates', 'drafts'] },
    ])
  })

  it('writes both items when two items are added one after the other', () => {
    const { getField, readWrites } = createListDetail({
      [PLUGIN_ID]: { ignoredFolders: ['templates'] },
    })

    getField().vm.$emit('add', 'drafts')
    getField().vm.$emit('add', 'archive/old')

    expect(readWrites().at(-1)).toStrictEqual({
      at: [PLUGIN_ID, 'ignoredFolders'],
      value: ['templates', 'drafts', 'archive/old'],
    })
  })

  it('writes the removal after an add on top of the added item', () => {
    const { getField, readWrites } = createListDetail({
      [PLUGIN_ID]: { ignoredFolders: ['templates'] },
    })

    getField().vm.$emit('add', 'drafts')
    getField().vm.$emit('remove', 'templates')

    expect(readWrites().at(-1)).toStrictEqual({
      at: [PLUGIN_ID, 'ignoredFolders'],
      value: ['drafts'],
    })
  })

  it('offers reset for a list changed from its default, and reset writes the default', async () => {
    const plugin = createPluginInfo(PLUGIN_ID, 'Tasks', {
      settings: [{ ...FOLDERS, default: ['templates'] }],
    })
    const { wrapper, readWrites } = createListDetail(
      { [PLUGIN_ID]: { ignoredFolders: ['templates', 'drafts'] } },
      plugin,
    )

    const resets = findResets(wrapper)
    expect(resets.map((one) => one.attributes('aria-label'))).toStrictEqual([
      'Reset Ignored folders to its default',
    ])
    await resets[0]!.trigger('click')

    expect(readWrites()).toStrictEqual([
      { at: [PLUGIN_ID, 'ignoredFolders'], value: ['templates'] },
    ])
  })

  it('offers no reset while the list holds its default', () => {
    const plugin = createPluginInfo(PLUGIN_ID, 'Tasks', {
      settings: [{ ...FOLDERS, default: ['templates'] }],
    })
    const { wrapper } = createListDetail({ [PLUGIN_ID]: { ignoredFolders: ['templates'] } }, plugin)

    expect(findResets(wrapper)).toHaveLength(0)
  })
})
