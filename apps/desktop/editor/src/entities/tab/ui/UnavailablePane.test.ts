import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { CATALOG_INCOMPATIBILITY, CATALOG_STATE } from '@numen/core-client'
import { createMockCatalogEntry, type MockCatalogEntryOptions } from '@numen/core-client/testing'
import { createAnsweredApi } from '@/testing/coreAnswers'
import { createDeferred } from '@/testing/deferred'
import { answerTabKinds } from '@/testing/tabCatalog'
import {
  createUnavailableKind,
  type UnavailableServices,
  type UnavailableState,
} from '../model/unavailable'
import UnavailablePane from './UnavailablePane.vue'

const notesOptions: MockCatalogEntryOptions = {
  name: 'Notes',
  feature: 'Markdown notes',
  tabKinds: ['reader:viewer'],
  state: CATALOG_STATE.DISABLED,
}

const mountPane = async (
  options: MockCatalogEntryOptions | null = notesOptions,
  services: Omit<UnavailableServices, 'findTabKindPlugin'> = {},
) => {
  const { findTabKindPlugin } = answerTabKinds(
    options === null ? [] : [createMockCatalogEntry('sample.notes', options)],
  )
  const kind = createUnavailableKind(UnavailablePane, 'reader:viewer', undefined, {
    ...services,
    findTabKindPlugin,
  })
  const state = kind.open('cards/Place.md', 'tab') as UnavailableState
  const view = mount(kind.pane, { props: { state } })
  await vi.waitFor(() => expect(state.lookup.value.status).not.toBe('pending'))
  await flushPromises()
  return view
}

describe('the pane of a tab whose plugin is disabled', () => {
  it('names the feature and the plugin from the catalog and omits the raw file path', async () => {
    const view = await mountPane()

    expect(view.text()).toContain('Plugin disabled')
    expect(view.text()).toContain('The plugin for Markdown notes is disabled.')
    expect(view.text()).not.toContain('cards/Place.md')
    expect(view.find('button').text()).toBe('Enable Notes')
  })

  it('enables the plugin the catalog names when the button is clicked', async () => {
    const enablePlugin = vi.fn(() => Promise.resolve())
    const view = await mountPane(notesOptions, { enablePlugin })

    await view.find('button').trigger('click')

    expect(enablePlugin).toHaveBeenCalledWith('sample.notes')
    expect(view.find('button').text()).toBe('Enabling Notes…')
    expect(view.find('button').attributes('disabled')).toBeDefined()
  })

  it('shows why enabling failed and offers the button again', async () => {
    const enablePlugin = vi.fn(() => Promise.reject(new Error('write failed')))
    const view = await mountPane(notesOptions, { enablePlugin })

    await view.find('button').trigger('click')
    await flushPromises()

    expect(view.find('[role="alert"]').exists()).toBe(true)
    expect(view.find('button').text()).toBe('Enable Notes')
  })
})

describe('the pane of a tab whose plugin is not installed', () => {
  it('offers to install it and opens the plugin on click', async () => {
    const openPlugin = vi.fn()
    const view = await mountPane(
      { ...notesOptions, state: CATALOG_STATE.AVAILABLE },
      { openPlugin },
    )

    expect(view.text()).toContain('Plugin not installed')
    expect(view.text()).toContain('The plugin for Markdown notes is not installed.')
    expect(view.find('button').text()).toBe('Install Notes')

    await view.find('button').trigger('click')

    expect(openPlugin).toHaveBeenCalledWith('sample.notes')
  })
})

describe('the pane of a tab whose plugin is installed but not running', () => {
  it('opens the plugin so the person can see why', async () => {
    const openPlugin = vi.fn()
    const view = await mountPane(
      { ...notesOptions, state: CATALOG_STATE.INSTALLED },
      { openPlugin },
    )

    expect(view.text()).toContain('Plugin not running')
    expect(view.find('button').text()).toBe('Open Settings')

    await view.find('button').trigger('click')

    expect(openPlugin).toHaveBeenCalledWith('sample.notes')
  })
})

describe('the pane of a tab whose plugin cannot run here', () => {
  it('states the requirement and offers no enable action', async () => {
    const enablePlugin = vi.fn(() => Promise.resolve())
    const view = await mountPane(
      {
        ...notesOptions,
        incompatibility: CATALOG_INCOMPATIBILITY.PROTOCOL,
        incompatibilityDetail: 'needs 2.0',
      },
      { enablePlugin },
    )

    expect(view.text()).toContain('Plugin incompatible')
    expect(view.text()).toContain('Notes cannot run in this application: needs 2.0')
    expect(view.find('button').text()).toBe('Open Settings')
    expect(enablePlugin).not.toHaveBeenCalled()
  })
})

describe('the pane of a tab no plugin in the catalog provides', () => {
  it('says the plugin is disabled or missing and opens the settings', async () => {
    const openPlugin = vi.fn()
    const view = await mountPane(null, { openPlugin })

    expect(view.text()).toContain('Plugin unavailable')
    expect(view.text()).toContain('The plugin for this tab is disabled or missing.')

    await view.find('button').trigger('click')

    expect(openPlugin).toHaveBeenCalledWith(undefined)
  })
})

describe('the pane while the catalog has not answered', () => {
  it('shows the loading state and no placeholder text until the answer arrives', async () => {
    const gate = createDeferred<void>()
    const { findTabKindPlugin } = answerTabKinds(
      [createMockCatalogEntry('sample.notes', notesOptions)],
      () => gate.promise,
    )
    const kind = createUnavailableKind(UnavailablePane, 'reader:viewer', undefined, {
      findTabKindPlugin,
    })
    const host = document.createElement('div')
    document.body.append(host)
    const texts: string[] = []
    const observer = new MutationObserver(() => texts.push(host.textContent ?? ''))
    observer.observe(host, { childList: true, subtree: true, characterData: true })
    const view = mount(kind.pane, {
      props: { state: kind.open('cards/Place.md', 'tab') },
      attachTo: host,
    })
    await flushPromises()

    expect(view.find('[aria-busy="true"]').exists()).toBe(true)
    expect(view.find('h3').exists()).toBe(false)
    expect(view.find('button').exists()).toBe(false)

    gate.answer()
    await vi.waitFor(() => expect(view.find('h3').exists()).toBe(true))
    observer.disconnect()

    expect(view.find('h3').text()).toBe('Plugin disabled')
    expect(view.find('button').text()).toBe('Enable Notes')
    expect(texts.length).toBeGreaterThan(0)
    expect(texts.filter((text) => text.includes('Plugin unavailable'))).toStrictEqual([])
    expect(texts.filter((text) => text.includes('Open Settings'))).toStrictEqual([])
    view.unmount()
    host.remove()
  })

  it('shows the missing-plugin text when the core cannot answer', async () => {
    const { findTabKindPlugin } = createAnsweredApi({})
    const kind = createUnavailableKind(UnavailablePane, 'reader:viewer', undefined, {
      findTabKindPlugin,
    })
    const view = mount(kind.pane, { props: { state: kind.open('cards/Place.md', 'tab') } })

    await vi.waitFor(() => expect(view.find('h3').exists()).toBe(true))

    expect(view.find('h3').text()).toBe('Plugin unavailable')
  })
})

describe('the placeholder while its plugin starts', () => {
  it('says the plugin is starting until its own plugin has started, and then what the catalog says of it', async () => {
    const starting = ref<ReadonlySet<string>>(new Set(['sample.notes', 'sample.other']))
    const view = await mountPane(notesOptions, {
      isPluginStarting: (id) => starting.value.has(id),
    })

    expect(view.find('[data-starting]').exists()).toBe(true)
    expect(view.text()).toContain('Starting the plugin')
    expect(view.text()).not.toContain('Plugin disabled')

    starting.value = new Set(['sample.other'])
    await flushPromises()

    expect(view.find('[data-starting]').exists()).toBe(false)
    expect(view.text()).toContain('Plugin disabled')
  })

  it('does not wait for other plugins that are still starting', async () => {
    const view = await mountPane(notesOptions, { isPluginStarting: (id) => id === 'sample.other' })

    expect(view.find('[data-starting]').exists()).toBe(false)
    expect(view.text()).toContain('Plugin disabled')
  })
})
