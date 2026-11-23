import { ref, shallowRef } from 'vue'
import { flushPromises } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import type { PluginInfo, PluginSettingInfo } from '@/entities/settings'
import type { JsonValue } from '@/shared/api/json'
import {
  createCatalogEntry,
  createInstallation,
  createPluginInfo,
  createSettingsState,
} from '../testing/installation'
import { appendPathEntry, useTextDrafts, removePathEntry } from './drafts'
import { openSelect } from './openSelect'
import { usePluginDetail, usePluginSwitch } from './usePluginDetail'
import { usePluginFields, usePluginPaths, useSettingDefaults } from './usePluginFields'
import { usePluginsList } from './usePluginsList'
import { useSettingsTab } from './useSettingsTab'
import { useSettingsViews } from './useSettingsViews'

describe('useTextDrafts', () => {
  it('holds the text typed at a key until it is written and preserves other keys', () => {
    const write = vi.fn()
    const text = useTextDrafts(write)
    text.setDraft('endpoint', 'http://127.0.0.1:9/v1')
    text.setDraft('model', 'gpt-4')
    expect(text.getDraft('endpoint')).toBe('http://127.0.0.1:9/v1')
    expect(text.getDraft('model')).toBe('gpt-4')
    expect(text.getDraft('other')).toBeUndefined()

    text.writeDraft('endpoint', 'http://127.0.0.1:9/v1')
    expect(write).toHaveBeenCalledWith('endpoint', 'http://127.0.0.1:9/v1')
    expect(text.getDraft('endpoint')).toBeUndefined()
    expect(text.getDraft('model')).toBe('gpt-4')
  })
})

describe('appendPathEntry and removePathEntry', () => {
  it('appends non-empty unique paths and trims existing entries', () => {
    expect(appendPathEntry('/opt/a', '')).toBe('/opt/a')
    expect(appendPathEntry('', '/opt/a')).toBe('/opt/a')
    expect(appendPathEntry('/opt/a', '/opt/a')).toBe('/opt/a')
    expect(appendPathEntry('  /opt/a \n\n /opt/b  ', '/opt/b')).toBe('/opt/a\n/opt/b')
    expect(appendPathEntry('  /opt/a \n\n /opt/b  ', '/opt/c')).toBe('/opt/a\n/opt/b\n/opt/c')
  })

  it('removes a specific path or clears all when targetPath is empty or omitted', () => {
    expect(removePathEntry('/opt/a\n/opt/b')).toBe('')
    expect(removePathEntry('/opt/a\n/opt/b', '')).toBe('')
    expect(removePathEntry('  /opt/a \n\n /opt/b  ', '/opt/a')).toBe('/opt/b')
    expect(removePathEntry('/opt/a\n/opt/b\n/opt/c', '/opt/b')).toBe('/opt/a\n/opt/c')
  })
})

describe('openSelect', () => {
  it('ignores null or undefined and focuses and dispatches ArrowDown on an element', () => {
    expect(() => openSelect(null)).not.toThrow()
    expect(() => openSelect(undefined)).not.toThrow()

    const button = document.createElement('button')
    document.body.append(button)
    const events: KeyboardEvent[] = []
    button.addEventListener('keydown', (event) => events.push(event))

    openSelect(button)

    expect(document.activeElement).toBe(button)
    expect(events).toHaveLength(1)
    expect(events[0]?.key).toBe('ArrowDown')
    expect(events[0]?.bubbles).toBe(true)
    expect(events[0]?.cancelable).toBe(true)
    button.remove()
  })
})

describe('usePluginDetail and usePluginSwitch', () => {
  it('tracks pending state, platform, and drops stale reads and deletions across plugin switches', async () => {
    const p1 = createPluginInfo('p1', 'Plugin One')
    const p2 = createPluginInfo('p2', 'Plugin Two')
    const currentPlugin = ref<PluginInfo>(p1)
    const platform = ref('darwin')

    const dataResolvers = new Map<string, (count: number) => void>()
    const readmeResolvers = new Map<string, (text: string) => void>()
    const deleteResolvers = new Map<string, (problem: string) => void>()

    const installation = createInstallation({
      getPlatform: () => platform.value,
      isPending: (at) => at[0] === 'p1' && at[1] === 'enabled',
      getPluginData: vi.fn(
        (id: string) =>
          new Promise<number>((resolve) => {
            dataResolvers.set(id, resolve)
          }),
      ),
      getReadme: vi.fn(
        (id: string) =>
          new Promise<string>((resolve) => {
            readmeResolvers.set(id, resolve)
          }),
      ),
      deletePluginData: vi.fn(
        (id: string) =>
          new Promise<string>((resolve) => {
            deleteResolvers.set(id, resolve)
          }),
      ),
    })

    const deps = {
      getSettings: () => installation.settings,
      getManager: () => installation.plugin,
      getPlugin: () => currentPlugin.value,
    }

    const sw = usePluginSwitch(deps)
    expect(sw.isPending.value).toBe(true)

    const detail = usePluginDetail(deps)
    expect(detail.isApple.value).toBe(true)
    expect(detail.deleteProblem.value).toBe('')
    expect(detail.readmeText.value).toBe('')
    expect(detail.fileCount.value).toBe(0)

    platform.value = 'linux'
    expect(detail.isApple.value).toBe(false)

    // Switch to p2 while p1 initial reads are still in flight
    const p1DataResolve = dataResolvers.get('p1')
    const p1ReadmeResolve = readmeResolvers.get('p1')
    currentPlugin.value = p2
    await flushPromises()

    // Resolve p2 first, then stale p1
    dataResolvers.get('p2')?.(4)
    readmeResolvers.get('p2')?.('Readme Two')
    await flushPromises()
    expect(detail.fileCount.value).toBe(4)
    expect(detail.readmeText.value).toBe('Readme Two')

    p1DataResolve?.(99)
    p1ReadmeResolve?.('Stale Readme One')
    await flushPromises()
    expect(detail.fileCount.value).toBe(4)
    expect(detail.readmeText.value).toBe('Readme Two')

    // Start deleteData on p2, then switch back to p1 before delete resolves
    detail.deleteData()
    currentPlugin.value = p1
    await flushPromises()
    expect(detail.readmeText.value).toBe('')
    expect(detail.fileCount.value).toBe(0)

    deleteResolvers.get('p2')?.('cannot delete p2')
    await flushPromises()
    expect(detail.deleteProblem.value).toBe('')
  })
})

describe('usePluginFields, useSettingDefaults, and usePluginPaths', () => {
  it('coerces setting types and falls back when stored values have wrong types', () => {
    const stored: Record<string, JsonValue | undefined> = {
      boolTrue: true,
      boolBad: 'yes',
      numBad: 'nan',
      strBad: 123,
    }
    const plugin = createPluginInfo('p1', 'P1', {
      settings: [
        { key: 'boolTrue', type: 'boolean', title: 'B1', description: '', default: false },
        { key: 'boolBad', type: 'boolean', title: 'B2', description: '', default: false },
        { key: 'numBad', type: 'number', title: 'N1', description: '', default: 7 },
        { key: 'strBad', type: 'string', title: 'S1', description: '', default: 'fallback' },
        { key: 'noDefault', type: 'string', title: 'ND', description: '', default: undefined },
      ],
    })
    const installation = createInstallation({
      getSetting: (at) => stored[at[1] ?? ''],
    })
    const deps = {
      getSettings: () => installation.settings,
      getHost: () => installation.plugin,
      getPlugin: () => plugin,
    }

    const fields = usePluginFields(deps)
    expect(fields.getBooleanSetting('boolTrue', false)).toBe(true)
    expect(fields.getBooleanSetting('boolBad', false)).toBe(false)
    expect(fields.getNumberSetting('numBad', 7)).toBe(7)
    expect(fields.getNumberSetting('missingNum', undefined)).toBe(0)
    expect(fields.getStringSetting('strBad', 'fallback')).toBe('fallback')
    expect(fields.getStringSetting('missingStr', undefined)).toBe('')

    const defaults = useSettingDefaults(deps)
    defaults.resetSetting({
      key: 'noDefault',
      type: 'string',
      title: 'ND',
      description: '',
      default: undefined,
    })
    expect(installation.settings.write).not.toHaveBeenCalled()
  })

  it('handles path selection guards, multiple file paths, and clearing problems', async () => {
    const multiSetting: PluginSettingInfo = {
      key: 'files',
      type: 'file',
      multiple: true,
      title: 'Files',
      description: '',
      default: undefined,
    }
    const folderSetting: PluginSettingInfo = {
      key: 'dir',
      type: 'string',
      multiple: true,
      title: 'Dir',
      description: '',
      default: '',
    }
    const currentId = ref('p1')
    const currentValue = shallowRef<JsonValue | undefined>('/opt/first')
    let resolveChoose: ((problem: string) => void) | undefined

    const installation = createInstallation({
      getSetting: () => currentValue.value,
      choosePath: vi.fn(
        () =>
          new Promise<string>((resolve) => {
            resolveChoose = resolve
          }),
      ),
    })
    const deps = {
      getSettings: () => installation.settings,
      getHost: () => installation.plugin,
      getPlugin: () => createPluginInfo(currentId.value, 'P', { settings: [multiSetting] }),
    }
    const fields = usePluginFields(deps)
    const paths = usePluginPaths(deps, fields)

    expect(paths.hasManyPaths(multiSetting)).toBe(true)
    expect(paths.hasManyPaths(folderSetting)).toBe(false)

    // When problem is non-empty, does not append or write
    paths.choosePath(multiSetting)
    resolveChoose?.('rejected')
    await flushPromises()
    expect(paths.problems.value).toStrictEqual({ files: 'rejected' })
    expect(installation.settings.write).not.toHaveBeenCalled()

    // When previous is empty and problem is empty, does not append
    currentValue.value = ''
    paths.choosePath(multiSetting)
    resolveChoose?.('')
    await flushPromises()
    expect(installation.settings.write).not.toHaveBeenCalled()

    // When previous is non-empty and new setting value is undefined, appends empty fallback cleanly
    currentValue.value = '/opt/first'
    paths.choosePath(multiSetting)
    currentValue.value = undefined
    resolveChoose?.('')
    await flushPromises()
    expect(installation.settings.write).toHaveBeenCalledWith([
      { at: ['p1', 'files'], value: '"/opt/first"' },
    ])

    // When plugin changes while choosePath is in flight, drops result
    paths.problems.value = { files: 'err-a', other: 'err-b' }
    paths.choosePath(multiSetting)
    currentId.value = 'p2'
    resolveChoose?.('late-error')
    await flushPromises()
    expect(paths.problems.value).toStrictEqual({ files: 'err-a', other: 'err-b' })

    // clearPath clears only the target setting's problem
    paths.clearPath(multiSetting)
    expect(paths.problems.value).toStrictEqual({ files: '', other: 'err-b' })
  })
})

describe('usePluginsList, useSettingsTab, and useSettingsViews', () => {
  it('defaults non-boolean enabled setting to true and avoids echo updates on query change', async () => {
    const query = ref('initial')
    const updateQuery = vi.fn((next: string) => {
      query.value = next
    })
    const installation = createInstallation({
      getSetting: () => 'invalid-boolean',
    })
    const p1 = createPluginInfo('p1', 'P1')
    const list = usePluginsList({
      getSettings: () => installation.settings,
      getCatalog: () => [],
      getQuery: () => query.value,
      listPlugins: () => [p1],
      updateQuery,
    })

    expect(list.enabled.value).toStrictEqual({ p1: true })

    // External query update should not call updateQuery back
    query.value = 'from-parent'
    await flushPromises()
    expect(list.searchQuery.value).toBe('from-parent')
    expect(updateQuery).not.toHaveBeenCalled()

    // Local searchQuery change calls updateQuery
    list.searchQuery.value = 'from-input'
    await flushPromises()
    expect(updateQuery).toHaveBeenCalledExactlyOnceWith('from-input')
  })

  it('tracks uninstalled catalog plugins, retains lastSelected plugin across empty reads, and filters markedSetting', () => {
    const p1 = createPluginInfo('p1', 'P1')
    const plugins = ref<readonly PluginInfo[]>([p1])
    const catalog = ref([
      createCatalogEntry('cat.one', { isAvailable: true }),
      createCatalogEntry('cat.two', { isAvailable: true }),
    ])
    const installation = createInstallation({ plugins, catalog })
    const state = createSettingsState(installation, {
      initialPluginId: ref<string | null>('p1'),
      markedSetting: ref({ pluginId: 'other', setting: 'source' }),
    })

    const tab = useSettingsTab(state)
    expect(tab.selectedPlugin.value).toStrictEqual(p1)
    expect(tab.marked.value).toBeNull()

    state.markedSetting.value = { pluginId: 'p1', setting: 'source' }
    expect(tab.marked.value).toStrictEqual({ pluginId: 'p1', setting: 'source' })

    // Empty read keeps lastSelected while selectedPluginId still matches
    plugins.value = []
    expect(tab.selectedPlugin.value).toStrictEqual(p1)

    // Switching to an uninstalled catalog entry clears selectedPlugin and finds uninstalled
    tab.selectPlugin('cat.two')
    expect(tab.selectedPlugin.value).toBeUndefined()
    expect(tab.uninstalled.value?.id).toBe('cat.two')

    const views = useSettingsViews({
      getSettings: () => installation.settings,
      getHost: () => installation.plugin,
      getPlugin: () => createPluginInfo('no-views', 'No Views'),
    })
    expect(views.views.value).toStrictEqual([])
  })
})
