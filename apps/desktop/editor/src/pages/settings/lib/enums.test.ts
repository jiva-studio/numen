import { describe, expect, it } from 'vitest'
import { createEnumFields } from './enums'
import { createPluginInfo } from '@/testing/installation'

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
})
