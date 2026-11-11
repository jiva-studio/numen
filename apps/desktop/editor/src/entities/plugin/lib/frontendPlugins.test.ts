import { describe, expect, it } from 'vitest'
import { getFrontendPluginIds } from './frontendPlugins'

describe('the plugins the window loads', () => {
  it('are the ones that declare a frontend, so a backend alone is never fetched', () => {
    const plugins = [
      { id: 'md.numen.agents.server', hasFrontend: false },
      { id: 'md.numen.views.sketch', hasFrontend: true },
    ]

    expect(getFrontendPluginIds(plugins)).toStrictEqual(['md.numen.views.sketch'])
  })
})
