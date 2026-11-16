/**
 * The tokens an editor is drawn with, written from the three editor settings. Every value here is
 * invented.
 */
import { describe, expect, it } from 'vitest'
import { readTokens } from '@/testing/readTokens'
import { getEditorCss } from './editorCss'

const SIZE = '--numen-editor-font-size'
const HEIGHT = '--numen-editor-line-height'
const FAMILY = '--numen-editor-font-family'

describe('the tokens of the editor written from its settings', () => {
  it('sets the font size in pixels and the line height as a multiplier of it', () => {
    const tokens = readTokens(getEditorCss({ fontSize: 20, lineHeight: 1.8, fontFamily: '' }))

    expect(tokens.get(SIZE)).toBe('20px')
    expect(tokens.get(HEIGHT)).toBe('1.8')
  })

  it('writes the size at the ends of its range as it is', () => {
    expect(readTokens(getEditorCss({ fontSize: 10, lineHeight: 1.2, fontFamily: '' })).get(SIZE)).toBe(
      '10px',
    )
    expect(readTokens(getEditorCss({ fontSize: 32, lineHeight: 2.4, fontFamily: '' })).get(HEIGHT)).toBe(
      '2.4',
    )
  })

  it('leaves the face to the theme where no family is chosen', () => {
    const tokens = readTokens(getEditorCss({ fontSize: 14, lineHeight: 1.6, fontFamily: '' }))

    expect(tokens.get(SIZE)).toBe('14px')
    expect(tokens.has(FAMILY)).toBe(false)
  })

  it('sets the face of the interface for Sans and the face of code for Mono', () => {
    const sans = readTokens(getEditorCss({ fontSize: 14, lineHeight: 1.6, fontFamily: 'sans' }))
    const mono = readTokens(getEditorCss({ fontSize: 14, lineHeight: 1.6, fontFamily: 'mono' }))

    expect(sans.get(FAMILY)).toContain('sans')
    expect(mono.get(FAMILY)).toContain('mono')
  })

  it('sets a serif face for Serif that is not a sans face', () => {
    const serif = readTokens(getEditorCss({ fontSize: 14, lineHeight: 1.6, fontFamily: 'serif' }))

    expect(serif.get(FAMILY)).toContain('serif')
    expect(serif.get(FAMILY)).not.toContain('sans')
  })

  it('sets a face for each choice that differs from the others', () => {
    const faces = ['sans', 'serif', 'mono'].map((choice) =>
      readTokens(getEditorCss({ fontSize: 14, lineHeight: 1.6, fontFamily: choice })).get(FAMILY),
    )

    expect(new Set(faces).size).toBe(3)
  })

  it.each(['cursive', 'Sans', 'sans; color: red', 'serif } body { display: none', 'mono /* x */', '<b>'])(
    'writes no family for the value %s that is none of the choices, and keeps the size',
    (value) => {
      const tokens = readTokens(getEditorCss({ fontSize: 18, lineHeight: 1.5, fontFamily: value }))

      expect(tokens.has(FAMILY)).toBe(false)
      expect(tokens.get(SIZE)).toBe('18px')
      expect([...tokens.keys()].sort()).toStrictEqual([HEIGHT, SIZE])
    },
  )
})
