import { describe, expect, it } from 'vitest'
import { createEnumFields } from './enums'
import { createPluginInfo } from '@/pages/settings/testing/installation'

describe('the fields of a plugin', () => {
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

      it('falls back to the value in force when no option is offered on the platform', () => {
        const noneAvailable = {
          key: 'k',
          type: 'enum' as const,
          title: 'K',
          description: '',
          default: 'windows',
          options: [
            { value: 'windows', title: 'Right', detail: '', when: "app.platform != 'darwin'" },
          ],
        }
        const fields = createEnumFields(
          () => plugin,
          () => undefined,
          () => false,
          () => 'darwin',
        )
        expect(fields.getEnumValue(noneAvailable)).toBe('windows')
      })

      it('formats undefined and null as empty text and non-string values as JSON', () => {
        const setting = {
          key: 'k',
          type: 'enum' as const,
          title: 'K',
          description: '',
          default: undefined,
          options: [],
        }
        const emptyFields = createEnumFields(
          () => plugin,
          () => undefined,
          () => false,
          () => 'linux',
        )
        const nullFields = createEnumFields(
          () => plugin,
          () => null,
          () => false,
          () => 'linux',
        )
        const numberFields = createEnumFields(
          () => plugin,
          () => 42,
          () => false,
          () => 'linux',
        )
        expect(emptyFields.getEnumValue(setting)).toBe('')
        expect(nullFields.getEnumValue(setting)).toBe('')
        expect(numberFields.getEnumValue(setting)).toBe('42')
      })
    })

    it('offers an option whose condition cannot be read', () => {
      const brokenSetting = {
        key: 'k',
        type: 'enum' as const,
        title: 'K',
        description: '',
        default: 'a',
        options: [{ value: 'a', title: 'A', detail: '', when: 'app.os ==' }],
      }
      const broken = createPluginInfo('p', 'P', {
        settings: [brokenSetting],
      })
      const fields = createEnumFields(
        () => broken,
        () => undefined,
        () => false,
        () => '',
      )
      expect(fields.getEnumChoices(brokenSetting)[0]?.isDisabled).toBeUndefined()
    })
  })
})
