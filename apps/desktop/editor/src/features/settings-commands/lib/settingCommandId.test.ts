import { describe, expect, it } from 'vitest'
import { getSettingCommandId, parseSettingCommandId } from './settingCommandId'

describe('the address of the command of a setting', () => {
  it('names the owner and the key path of the setting', () => {
    expect(getSettingCommandId('app', 'appearance.theme')).toBe('setting:app:appearance.theme')
    expect(getSettingCommandId('md.example.scanner', 'autoScan')).toBe(
      'setting:md.example.scanner:autoScan',
    )
  })

  it('is read back into the owner and the key, dots of a plugin id included', () => {
    expect(parseSettingCommandId('setting:md.example.scanner:autoScan')).toStrictEqual({
      owner: 'md.example.scanner',
      key: 'autoScan',
    })
    expect(parseSettingCommandId('setting:app:appearance.theme')).toStrictEqual({
      owner: 'app',
      key: 'appearance.theme',
    })
  })

  it('gives no owner for the id of another command', () => {
    expect(parseSettingCommandId('app:note.create')).toBeNull()
    expect(parseSettingCommandId('setting::key')).toBeNull()
    expect(parseSettingCommandId('setting:app:')).toBeNull()
  })
})
