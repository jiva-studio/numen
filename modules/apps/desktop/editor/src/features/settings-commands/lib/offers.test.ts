import { describe, expect, it } from 'vitest'
import type { Ranges, Sizes, Theme } from '@/entities/settings'
import { WORDS as words } from '@/shared/words'
import { INTERFACE_SCALE, TEXT_SCALE } from './appearanceValues'
import { getModeGroups, getSizeGroups, getThemeGroups } from './offers'

const THEMES: readonly Theme[] = [
  { name: 'preset:numen', title: 'numen', isBuiltIn: true, isPinned: false },
  { name: 'preset:dracula', title: 'dracula', isBuiltIn: true, isPinned: true },
  { name: 'mine:sea', title: 'sea', isBuiltIn: false, isPinned: false },
]

const SIZES: Sizes = { interfaceScale: 1, textScale: 1 }
const BOUNDS: Ranges = {
  interfaceScale: { least: 0.8, most: 2 },
  textScale: { least: 0.8, most: 1.75 },
}

describe('the themes offered', () => {
  it('stands the shelf the theme worn came off first', () => {
    expect(getThemeGroups(THEMES, 'mine:sea', words).map((one) => one.id)).toStrictEqual([
      'owned',
      'shipping',
    ])
    expect(getThemeGroups(THEMES, 'preset:numen', words).map((one) => one.id)).toStrictEqual([
      'shipping',
      'owned',
    ])
  })

  it('opens the shelf on the theme in force', () => {
    const shipping = getThemeGroups(THEMES, 'preset:dracula', words)[0]

    expect(shipping?.items[0]?.title).toBe('dracula')
    expect(shipping?.items[0]?.inForce).toBe(true)
  })
})

describe('the modes offered', () => {
  it('marks the one in force and leaves the rest unsaid', () => {
    const items = getModeGroups('dark', false, words).flatMap((one) => one.items)

    expect(items.filter((one) => one.inForce).map((one) => one.title)).toStrictEqual([words.dark])
    expect(items.some((one) => one.disabled)).toBe(false)
  })

  it('says why none can be chosen where the theme worn pins the two halves', () => {
    const items = getModeGroups('dark', true, words).flatMap((one) => one.items)

    expect(items.every((one) => one.disabled)).toBe(true)
    expect(items.every((one) => one.detail === words.pinned)).toBe(true)
  })
})

describe('the sizes offered', () => {
  const titles = (command: string, text = '') =>
    getSizeGroups(command, text, SIZES, BOUNDS, words)
      .flatMap((one) => one.items)
      .map((one) => one.title)

  it('goes as far as the range of that command reaches', () => {
    expect(titles(INTERFACE_SCALE).at(-1)).toBe('200%')
    expect(titles(TEXT_SCALE).at(-1)).toBe('175%')
  })

  it('stands a number a person typed in its place, once', () => {
    expect(titles(INTERFACE_SCALE, '137')).toContain('137%')
    expect(titles(INTERFACE_SCALE, '137').indexOf('137%')).toBe(6)
    expect(titles(INTERFACE_SCALE, '150').filter((one) => one === '150%')).toHaveLength(1)
  })

  it('leaves out a number the range does not reach', () => {
    expect(titles(INTERFACE_SCALE, '250')).not.toContain('250%')
  })
})
