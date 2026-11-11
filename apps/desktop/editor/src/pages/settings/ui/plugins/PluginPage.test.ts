/**
 * The page of one plugin, opened after the page of another while the first page's reads are still
 * on their way. Every value here is invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

import { getSettingAt as at } from '@/pages/settings/testing/settingAt'
import {
  createInstallation,
  createPluginInfo,
  createSettingsState,
} from '@/pages/settings/testing/installation'
import SettingsTab from '../SettingsTab.vue'

const PATH_SETTING = {
  key: 'executablePath',
  type: 'file' as const,
  title: 'Executable Path',
  description: '',
  default: '',
}

const FIRST = createPluginInfo('md.example.first', 'First Agent', { settings: [PATH_SETTING] })
const SECOND = createPluginInfo('md.example.second', 'Second Agent', { settings: [PATH_SETTING] })

const FILE = {
  [FIRST.id]: { executablePath: '/opt/first/bin/first-cli' },
  [SECOND.id]: { executablePath: '/opt/second/bin/second-cli' },
}

const createHeld = <T>() => {
  const answers = new Map<string, (value: T) => void>()
  const call = (pluginId: string) =>
    new Promise<T>((resolve) => {
      answers.set(pluginId, resolve)
    })
  const answer = (pluginId: string, value: T) => answers.get(pluginId)?.(value)
  return { call, answer }
}

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

describe('the page of a plugin opened after another', () => {
  it('never shows a value of the plugin shown before', async () => {
    const fileCounts = createHeld<number>()
    const readmes = createHeld<string>()
    vi.stubGlobal('fetch', async (url: string) => {
      const pluginId = url.split('/')[2] ?? ''
      const text = await readmes.call(pluginId)
      return { ok: true, headers: new Headers(), text: async () => text }
    })
    const initialPluginId = ref<string | null>(FIRST.id)
    const installation = createInstallation({
      getSetting: (path) => at(FILE, path),
      plugins: ref([FIRST, SECOND]),
      getPluginData: (pluginId) => fileCounts.call(pluginId),
    })
    const wrapper = mount(SettingsTab, {
      props: { state: createSettingsState(installation, { initialPluginId }) },
      attachTo: document.body,
    })
    const history: string[] = []
    const record = () => {
      const page = wrapper.element as HTMLElement
      if (!page.textContent?.includes(SECOND.name)) return
      history.push(page.textContent ?? '')
      for (const one of page.querySelectorAll('[title]'))
        history.push(one.getAttribute('title') ?? '')
    }
    const observer = new MutationObserver(record)
    observer.observe(wrapper.element, { subtree: true, childList: true, characterData: true })

    await flushPromises()
    initialPluginId.value = SECOND.id
    await nextTick()
    record()
    fileCounts.answer(SECOND.id, 2)
    readmes.answer(SECOND.id, 'Second agent readme.')
    await flushPromises()
    fileCounts.answer(FIRST.id, 7)
    readmes.answer(FIRST.id, 'First agent readme.')
    await flushPromises()
    observer.disconnect()

    const text = history.join('\n')
    expect(text).toContain('second-cli')
    expect(text).toContain('Second agent readme.')
    expect(text).toContain('2 files')
    expect(text).not.toContain('first-cli')
    expect(text).not.toContain('First agent readme.')
    expect(text).not.toContain('7 files')
    wrapper.unmount()
  })
})
