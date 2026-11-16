import { describe, expect, it } from 'vitest'
import { getPickerState, getTagKey } from './pickerState'

describe('getTagKey', () => {
  it('ignores case, the spaces around a tag, a leading hash and the spaces around a segment', () => {
    expect(getTagKey(' #Projects / Web ')).toBe('projects/web')
  })

  it('joins the Unicode forms of one letter', () => {
    expect(getTagKey('Café')).toBe(getTagKey('Café'))
  })
})

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
