/**
 * The settings the manifest declares for the release feed: the address is asked for under the
 * Custom channel alone.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

interface Setting {
  readonly key: string
  readonly default?: unknown
  readonly options?: readonly { readonly value: string }[]
  readonly when?: { readonly setting: string; readonly is: readonly string[] }
}

const { settings } = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as { settings: readonly Setting[] }

const findSetting = (key: string): Setting => {
  const found = settings.find((one) => one.key === key)
  if (!found) throw new Error(`the manifest declares no setting ${key}`)
  return found
}

describe('the release feed address setting', () => {
  it('is shown under the Custom channel and under no other', () => {
    const channel = findSetting('channel')
    const { when } = findSetting('endpoint')

    expect(channel.options?.map((one) => one.value)).toStrictEqual(['stable', 'beta', 'custom'])
    expect(when).toStrictEqual({ setting: 'channel', is: ['custom'] })
  })

  it('is empty until one is typed', () => {
    expect(findSetting('endpoint').default).toBe('')
  })
})
