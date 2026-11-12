import { describe, expect, it } from 'vitest'
import type { SlotItem } from '@numen/plugin-sdk'
import { useIsDrawn } from './useIsDrawn'

const createGap = (key: string): SlotItem => ({ kind: 'separator', key }) as SlotItem
const getKey = (item: SlotItem): string => String(item.key)

describe('whether an entry of a slot is drawn', () => {
  it('is true for an entry whose key is not hidden', () => {
    const isDrawn = useIsDrawn(
      () => getKey,
      () => new Set(['b']),
    )

    expect(isDrawn(createGap('a'))).toBe(true)
  })

  it('is false for an entry whose key is hidden', () => {
    const isDrawn = useIsDrawn(
      () => getKey,
      () => new Set(['b']),
    )

    expect(isDrawn(createGap('b'))).toBe(false)
  })
})
