import { describe, expect, it } from 'vitest'
import type { PluginSettingInfo } from '@/entities/settings'
import { getSettingSections } from './settingSections'

const setting = (key: string, group?: string): PluginSettingInfo => ({
  key,
  type: 'boolean',
  title: key,
  description: '',
  default: false,
  ...(group ? { group } : {}),
})

const GROUPS = [
  { id: 'look', title: 'Look' },
  { id: 'feel', title: 'Feel' },
]

const keys = (settings: readonly PluginSettingInfo[]) => settings.map((one) => one.key)

describe('getSettingSections', () => {
  it('keeps every setting plain while the plugin declares no group', () => {
    const sections = getSettingSections([setting('a'), setting('b')], undefined)

    expect(keys(sections.plain)).toStrictEqual(['a', 'b'])
    expect(sections.islands).toStrictEqual([])
  })

  it('draws the islands in the order of the groups, whatever the order of the settings', () => {
    const sections = getSettingSections(
      [setting('a', 'feel'), setting('b', 'look'), setting('c', 'feel')],
      GROUPS,
    )

    expect(sections.islands.map((one) => one.title)).toStrictEqual(['Look', 'Feel'])
    expect(keys(sections.islands[1]?.settings ?? [])).toStrictEqual(['a', 'c'])
  })

  it('leaves a setting without a group in the plain island', () => {
    const sections = getSettingSections([setting('a', 'look'), setting('b')], GROUPS)

    expect(keys(sections.plain)).toStrictEqual(['b'])
  })

  it('draws a setting of a group nobody declared in the plain island', () => {
    const sections = getSettingSections([setting('a', 'gone')], GROUPS)

    expect(keys(sections.plain)).toStrictEqual(['a'])
    expect(sections.islands).toStrictEqual([])
  })

  it('leaves out a group that holds no setting', () => {
    const sections = getSettingSections([setting('a', 'look')], GROUPS)

    expect(sections.islands.map((one) => one.id)).toStrictEqual(['look'])
  })
})
