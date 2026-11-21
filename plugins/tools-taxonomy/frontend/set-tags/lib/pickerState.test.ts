import { describe, expect, it } from 'vitest'
import { getPickerState } from './pickerState'

describe('getPickerState', () => {
  it('is set when every note has the tag', () => {
    expect(getPickerState('Port', [['Port'], ['Sea', 'port']])).toBe('set')
  })

  it('is mixed when only some notes have the tag', () => {
    expect(getPickerState('Port', [['Port'], []])).toBe('mixed')
  })

  it('is unset when no note has the tag', () => {
    expect(getPickerState('Port', [['Sea'], []])).toBe('unset')
  })
})
