import { describe, expect, it } from 'vitest'
import { createMockPickerHost } from '@numen/plugin-sdk/testing'
import type { Field } from '../api/templates'
import { collectFieldValues } from './collectFieldValues'

const OWNER: Field = { kind: 'text', label: 'Owner', options: [], default: 'Ann' }
const LEVEL: Field = { kind: 'select', label: 'Level', options: ['low', 'high'], default: 'high' }

const tick = () => new Promise((resolve) => setTimeout(resolve))

describe('asking the fields in the palette', () => {
  it('asks each field in order, named in the title, and answers by label', async () => {
    const palette = createMockPickerHost()
    const asking = collectFieldValues(palette, 'Kickoff', [OWNER, LEVEL])

    expect(palette.getQuestion()).toStrictEqual({ title: 'Kickoff · Owner', value: 'Ann' })
    palette.answer('Bob')
    await tick()
    expect(palette.getQuestion()).toStrictEqual({
      title: 'Kickoff · Level',
      options: [
        { value: 'high', title: 'high' },
        { value: 'low', title: 'low' },
      ],
    })
    palette.answer('low')

    await expect(asking).resolves.toStrictEqual({ Owner: 'Bob', Level: 'low' })
  })

  it('offers the options as given when the default is none of them', async () => {
    const palette = createMockPickerHost()
    const asking = collectFieldValues(palette, 'T', [{ ...LEVEL, default: '' }])

    expect(palette.getQuestion()).toMatchObject({
      options: [
        { value: 'low', title: 'low' },
        { value: 'high', title: 'high' },
      ],
    })
    palette.dismiss()
    await asking
  })

  it('stops at the question the person dismisses', async () => {
    const palette = createMockPickerHost()
    const asking = collectFieldValues(palette, 'Kickoff', [OWNER, LEVEL])

    palette.dismiss()

    await expect(asking).resolves.toBeUndefined()
    expect(palette.getQuestion()).toBeUndefined()
  })
})
