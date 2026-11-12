import { describe, expect, it } from 'vitest'
import { createEnumFields } from './enums'
import { createPluginInfo } from '@/pages/settings/testing/installation'

describe('the fields of a plugin', () => {
  it('places secrets after their when.setting, provider, baseUrl, or at the top', () => {
    const withProvider = createPluginInfo('p', 'P', {
      settings: [
        { key: 'provider', type: 'string', title: 'P', description: '', default: 'a' },
        { key: 'baseUrl', type: 'string', title: 'B', description: '', default: 'b' },
      ],
      secrets: [
        { name: 's1', title: 'S1', description: '', group: '' },
        {
          name: 's2',
          title: 'S2',
          description: '',
          group: '',
          when: { setting: 'baseUrl', is: ['b'] },
        },
      ],
    })
    const fields1 = createEnumFields(
      () => withProvider,
      () => undefined,
    )
    expect(fields1.getTopSecrets()).toStrictEqual([])
    expect(fields1.getSecretsAfter('provider').map((s) => s.name)).toStrictEqual(['s1'])
    expect(fields1.getSecretsAfter('baseUrl').map((s) => s.name)).toStrictEqual(['s2'])

    const withBaseUrlOnly = createPluginInfo('p', 'P', {
      settings: [{ key: 'baseUrl', type: 'string', title: 'B', description: '', default: 'b' }],
      secrets: [{ name: 's1', title: 'S1', description: '', group: '' }],
    })
    const fields2 = createEnumFields(
      () => withBaseUrlOnly,
      () => undefined,
    )
    expect(fields2.getTopSecrets()).toStrictEqual([])
    expect(fields2.getSecretsAfter('baseUrl').map((s) => s.name)).toStrictEqual(['s1'])

    const withNeither = createPluginInfo('p', 'P', {
      settings: [{ key: 'other', type: 'string', title: 'O', description: '', default: 'o' }],
      secrets: [{ name: 's1', title: 'S1', description: '', group: '' }],
    })
    const fields3 = createEnumFields(
      () => withNeither,
      () => undefined,
    )
    expect(fields3.getTopSecrets().map((s) => s.name)).toStrictEqual(['s1'])
    expect(fields3.getSecretsAfter('')).toStrictEqual([])
  })

  describe('the choices of an enum whose options carry a condition', () => {
    const plugin = createPluginInfo('p', 'P', {
      settings: [
        {
          key: 'buttons',
          type: 'enum',
          title: 'Buttons',
          description: '',
          default: 'mac',
          options: [
            { value: 'mac', title: 'Left', detail: '' },
            { value: 'windows', title: 'Right', detail: '', when: "app.platform != 'darwin'" },
            { value: 'none', title: 'None', detail: '', when: "app.platform != 'darwin'" },
          ],
        },
      ],
    })
    const getChoices = (platform: string) => {
      const fields = createEnumFields(
        () => plugin,
        () => undefined,
        () => false,
        () => platform,
      )
      return fields.getEnumChoices(plugin.settings[0]!).map((one) => [one.id, one.isDisabled])
    }

    it.each(['linux', 'windows'])('offers all three on %s', (platform) => {
      expect(getChoices(platform)).toStrictEqual([
        ['mac', undefined],
        ['windows', undefined],
        ['none', undefined],
      ])
    })

    it('offers only Left on macOS and disables the others', () => {
      expect(getChoices('darwin')).toStrictEqual([
        ['mac', undefined],
        ['windows', true],
        ['none', true],
      ])
    })

    describe('the value shown', () => {
      const getShownOptions = (platform: string, stored?: string) => {
        const buttons = { ...plugin.settings[0]!, default: 'windows' }
        const fields = createEnumFields(
          () => plugin,
          () => stored,
          () => false,
          () => platform,
        )
        return fields.getEnumValue(buttons)
      }

      it('is Left on macOS where the default is Right', () => {
        expect(getShownOptions('darwin')).toBe('mac')
      })

      it('is Left on macOS where the stored value is Right or None', () => {
        expect(getShownOptions('darwin', 'windows')).toBe('mac')
        expect(getShownOptions('darwin', 'none')).toBe('mac')
      })

      it.each(['linux', 'windows'])('stays Right by default on %s', (platform) => {
        expect(getShownOptions(platform)).toBe('windows')
      })

      it('is a stored value that is offered, as stored', () => {
        expect(getShownOptions('darwin', 'mac')).toBe('mac')
        expect(getShownOptions('linux', 'none')).toBe('none')
      })
    })

    it('offers an option whose condition cannot be read', () => {
      const broken = createPluginInfo('p', 'P', {
        settings: [
          {
            key: 'k',
            type: 'enum',
            title: 'K',
            description: '',
            default: 'a',
            options: [{ value: 'a', title: 'A', detail: '', when: 'app.os ==' }],
          },
        ],
      })
      const fields = createEnumFields(
        () => broken,
        () => undefined,
      )
      expect(fields.getEnumChoices(broken.settings[0]!)[0]?.isDisabled).toBeUndefined()
    })
  })
})
