import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { CATALOG_INCOMPATIBILITY, CATALOG_STATE } from '@numen/plugin-sdk'
import {
  createMockCatalogEntry,
  createMockPluginCatalog,
  type MockCatalogEntryOptions,
} from '@numen/plugin-sdk/testing'
import { createUnavailableKind, type UnavailableServices } from './unavailable'

const notesOptions: MockCatalogEntryOptions = {
  name: 'Notes',
  feature: 'Markdown notes',
  tabKinds: ['reader:viewer'],
  state: CATALOG_STATE.DISABLED,
}

const createServices = (
  options: MockCatalogEntryOptions | null = notesOptions,
  extra: Partial<UnavailableServices> = {},
): UnavailableServices => ({
  catalog: createMockPluginCatalog(
    options === null ? [] : [createMockCatalogEntry('sample.notes', options)],
  ),
  ...extra,
})

const mountPane = async (services: UnavailableServices, kindName = 'reader:viewer') => {
  const kind = createUnavailableKind(kindName, undefined, services)
  const view = mount(kind.pane, { props: { state: kind.open('cards/Place.md', 'tab') } })
  await flushPromises()
  return view
}

describe('createUnavailableKind', () => {
  it('keeps the name of the missing kind and what the tab stood on', () => {
    const kind = createUnavailableKind('reader:viewer')

    const state = kind.open('/a.bin', 'tab')

    expect(kind.kind).toBe('reader:viewer')
    expect(kind.getOpenTab?.(state)).toEqual({ path: '/a.bin' })
    expect(kind.identity?.('/a.bin')).toBe('/a.bin')
  })

  it('keeps the state the plugin saved for the tab', () => {
    const kind = createUnavailableKind('reader:viewer')

    const state = kind.open('a.bin', 'tab', { page: 4 })

    expect(kind.getState?.(state)).toEqual({ page: 4 })
    expect(kind.getTitle?.(state)).toBe('a.bin')
  })

  it('keeps the original tab title when provided', () => {
    const kind = createUnavailableKind('files', { title: 'Files' })

    expect(kind.getTitle?.(kind.open('', 'tab'))).toBe('Files')
  })

  it('names a tab with no path and no title after the kind the catalog declares once it answers', async () => {
    const catalog = createMockPluginCatalog([
      createMockCatalogEntry('sample.decks', { tabKinds: [{ kind: 'deck', title: 'Deck' }] }),
    ])
    const kind = createUnavailableKind('deck', undefined, { catalog })
    const state = kind.open('', 'tab')
    expect(kind.getTitle?.(state)).toBe('deck')

    await flushPromises()

    expect(kind.getTitle?.(state)).toBe('Deck')
    expect(catalog.getAskedKinds()).toStrictEqual(['deck'])
  })
})

describe('the pane of a tab whose plugin is disabled', () => {
  it('names the feature and the plugin from the catalog and omits the raw file path', async () => {
    const view = await mountPane(createServices())

    expect(view.text()).toContain('Plugin disabled')
    expect(view.text()).toContain('The plugin for Markdown notes is disabled.')
    expect(view.text()).not.toContain('cards/Place.md')
    expect(view.find('button').text()).toBe('Enable Notes')
  })

  it('enables the plugin the catalog names when the button is clicked', async () => {
    const enablePlugin = vi.fn(() => Promise.resolve())
    const view = await mountPane(createServices(notesOptions, { enablePlugin }))

    await view.find('button').trigger('click')

    expect(enablePlugin).toHaveBeenCalledWith('sample.notes')
    expect(view.find('button').text()).toBe('Enabling Notes…')
    expect(view.find('button').attributes('disabled')).toBeDefined()
  })

  it('shows why enabling failed and offers the button again', async () => {
    const enablePlugin = vi.fn(() => Promise.reject(new Error('write failed')))
    const view = await mountPane(createServices(notesOptions, { enablePlugin }))

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
      createServices({ ...notesOptions, state: CATALOG_STATE.AVAILABLE }, { openPlugin }),
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
      createServices({ ...notesOptions, state: CATALOG_STATE.INSTALLED }, { openPlugin }),
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
      createServices(
        {
          ...notesOptions,
          incompatibility: CATALOG_INCOMPATIBILITY.APP_TOO_OLD,
          incompatibilityDetail: 'needs 2.0',
        },
        { enablePlugin },
      ),
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
    const view = await mountPane(createServices(null, { openPlugin }))

    expect(view.text()).toContain('Plugin unavailable')
    expect(view.text()).toContain('The plugin for this tab is disabled or missing.')

    await view.find('button').trigger('click')

    expect(openPlugin).toHaveBeenCalledWith(undefined)
  })

  it('is the same without a catalog', async () => {
    const view = await mountPane({})

    expect(view.text()).toContain('Plugin unavailable')
  })
})
