import { describe, expect, it } from 'vitest'
import {
  INTERFACE_SCALE,
  MODE,
  TEXT_SCALE,
  getModeId,
  getSizeId,
  its,
  modeOf,
  onto,
  sizeOf,
} from './appearanceValues'

describe('the name a mode is offered under', () => {
  it('names the command it belongs to, and reads back as that mode', () => {
    expect(getModeId('dark')).toBe(`${MODE}:dark`)
    expect(modeOf(getModeId('dark'))).toBe('dark')
  })

  it('is nothing for a row naming a theme', () => {
    expect(modeOf('mine:sea')).toBeNull()
    expect(modeOf('')).toBeNull()
  })
})

describe('the name a size is offered under', () => {
  it('names which of the two it is, and reads back as that size', () => {
    expect(getSizeId(TEXT_SCALE, 1.25)).toBe(`${TEXT_SCALE}:1.25`)
    expect(sizeOf(`${TEXT_SCALE}:1.25`)).toStrictEqual({ which: TEXT_SCALE, size: 1.25 })
  })

  it('is nothing for a row of any other command, and for a size of none', () => {
    expect(sizeOf('mine:sea')).toBeNull()
    expect(sizeOf(`${INTERFACE_SCALE}:0`)).toBeNull()
    expect(sizeOf(`${INTERFACE_SCALE}:large`)).toBeNull()
  })
})

describe('the pair of sizes', () => {
  const both = { interfaceScale: 1, textScale: 1.5 }

  it('says what is held of one of the two', () => {
    expect(its(both, TEXT_SCALE)).toBe(1.5)
  })

  it('puts one in its place and leaves the other where it was', () => {
    expect(onto(both, INTERFACE_SCALE, 1.25)).toStrictEqual({
      interfaceScale: 1.25,
      textScale: 1.5,
    })
    expect(both.interfaceScale).toBe(1)
  })
})
