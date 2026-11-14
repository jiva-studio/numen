import { describe, expect, it } from 'vitest'
import { shallowRef } from 'vue'
import { NO_TAB_KIND_PLUGINS } from '@/testing/tabs'
import type { TabKindPlugin, UnavailableLookup } from '../lib/unavailableView'
import type { UnavailableServices, UnavailableState } from './unavailable'
import { useIsStartingPlugin } from './useIsStartingPlugin'

const createState = (
  lookup: UnavailableLookup,
  services: UnavailableServices,
): UnavailableState => ({
  kind: 'reader:viewer',
  targetPath: 'cards/Place.md',
  saved: undefined,
  lookup: shallowRef(lookup),
  services,
})

const found = (id: string): UnavailableLookup => {
  const plugin: TabKindPlugin = {
    id,
    name: 'Notes',
    feature: 'Markdown notes',
    standing: 'installed',
    incompatibility: null,
    kindTitle: '',
  }
  return { status: 'found', plugin }
}

describe('whether the plugin of an unavailable tab is starting', () => {
  it('is true when the window reports the found plugin as starting', () => {
    const state = createState(found('sample.notes'), {
      ...NO_TAB_KIND_PLUGINS,
      isPluginStarting: (id) => id === 'sample.notes',
    })

    expect(useIsStartingPlugin(() => state).value).toBe(true)
  })

  it('is false when the window does not report the plugin as starting', () => {
    const state = createState(found('sample.notes'), {
      ...NO_TAB_KIND_PLUGINS,
      isPluginStarting: () => false,
    })

    expect(useIsStartingPlugin(() => state).value).toBe(false)
  })

  it('is false while the catalog has not answered', () => {
    const state = createState({ status: 'pending' }, {
      ...NO_TAB_KIND_PLUGINS,
      isPluginStarting: () => true,
    })

    expect(useIsStartingPlugin(() => state).value).toBe(false)
  })

  it('is false when the window cannot tell', () => {
    const state = createState(found('sample.notes'), NO_TAB_KIND_PLUGINS)

    expect(useIsStartingPlugin(() => state).value).toBe(false)
  })
})
