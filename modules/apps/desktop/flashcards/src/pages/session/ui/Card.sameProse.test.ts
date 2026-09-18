// @vitest-environment jsdom
/**
 * A card face is written once and drawn in two windows: here, and in the
 * editor's stencil pane. Both draw it through `CardProse`, so the markup a face
 * comes to is the same in both, and this is what holds them to it.
 */
import { CardProse } from '@numen/ui'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import Card from './Card.vue'

/** What each of the faces below has to survive being drawn. */
const FACES = [
  '<p>plain prose</p>',
  '<p>a <strong>strong</strong> word and an <em>emphasised</em> one</p>',
  '<ul><li>one</li><li>two</li></ul>',
  '<p>a <a href="numen://note/01ABC">link into the vault</a></p>',
  '<p>an image: <img src="data:image/png;base64,iVBORw0KGgo=" alt="a plate" /></p>',
  '<blockquote><p>what was said</p></blockquote>',
  '<p>a <script>alert(1)</script> that must not survive</p>',
  '<p>Devanagari: कृष्ण, and IAST: kṛṣṇa</p>',
  '<table><tr><td>a cell</td></tr></table>',
  '',
]

/** The prose of a face as the review window draws it. */
const inTheReviewWindow = (text: string): string => {
  const drawn = mount(Card, { props: { front: text, back: '', isShown: false } })
  const said = drawn.find('.prose').element.innerHTML
  drawn.unmount()
  return said
}

/** The prose of the same face as the editor's pane draws it. */
const inTheEditor = (text: string): string => {
  const drawn = mount(CardProse, { props: { text } })
  const said = drawn.find('.prose').element.innerHTML
  drawn.unmount()
  return said
}

describe('one face, two windows', () => {
  it('comes to the same markup in both', () => {
    for (const face of FACES) {
      expect(inTheReviewWindow(face)).toBe(inTheEditor(face))
    }
  })

  // The control: the faces above are not all empty, so the comparison above is
  // comparing something.
  it('draws what it was given', () => {
    expect(inTheReviewWindow('<p>plain prose</p>')).toContain('plain prose')
    expect(
      inTheReviewWindow('<p>a <script>alert(1)</script> that must not survive</p>'),
    ).not.toContain('alert(1)')
  })
})
