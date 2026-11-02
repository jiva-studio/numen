import { describe, expect, it } from 'vitest'
import { toThemeChoices } from './enums'

const LABELS = {
  shipped: 'Built-in',
  owned: 'Custom',
  getPluginName: (pluginId: string) =>
    pluginId === 'org.example.themes' ? 'Sample Themes' : pluginId,
}

describe('the themes a select offers', () => {
  it('are grouped as shipped, by plugin name, and owned, in that order', () => {
    const choices = toThemeChoices(
      [
        { name: 'mine:sea', title: 'sea', isBuiltIn: false, pluginId: '' },
        {
          name: 'plugin:org.example.themes/night',
          title: 'Night',
          isBuiltIn: false,
          pluginId: 'org.example.themes',
        },
        { name: 'preset:numen', title: 'numen', isBuiltIn: true, pluginId: '' },
      ],
      LABELS,
    )

    expect(choices).toStrictEqual([
      { id: 'preset:numen', text: 'numen', group: 'Built-in' },
      { id: 'plugin:org.example.themes/night', text: 'Night', group: 'Sample Themes' },
      { id: 'mine:sea', text: 'sea', group: 'Custom' },
    ])
  })
})
