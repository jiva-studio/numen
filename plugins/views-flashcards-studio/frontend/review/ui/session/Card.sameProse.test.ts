// @vitest-environment jsdom
/**
 * A review card is written once and drawn in two windows: here, and in the
 * editor's card model pane. Both render it through `renderCardHtml`, so the markup a review card
 * evaluates to is the same in both, and this is what holds them to it.
 */
import { CardProse } from '../../../editor/ui/cards/card-prose'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import Card from './Card.vue'

/** What each of the review cards below has to survive being drawn. */
const REVIEW_CARDS = [
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

const renderInReviewWindow = (text: string): string => {
  const renderedElement = mount(Card, { props: { front: text, back: '', isShown: false } })
  const response = renderedElement.find('.prose').element.innerHTML
  renderedElement.unmount()
  return response
}

const renderInEditor = (text: string): string => {
  const renderedElement = mount(CardProse, { props: { text } })
  const response = renderedElement.find('.prose').element.innerHTML
  renderedElement.unmount()
  return response
}

describe('one review card, two windows', () => {
  it('evaluates to the same markup in both', () => {
    for (const reviewCard of REVIEW_CARDS) {
      expect(renderInReviewWindow(reviewCard)).toBe(renderInEditor(reviewCard))
    }
  })

  // The control: the review cards above are not all empty, so the comparison above is
  // comparing something.
  it('draws what it was given', () => {
    expect(renderInReviewWindow('<p>plain prose</p>')).toContain('plain prose')
    expect(
      renderInReviewWindow('<p>a <script>alert(1)</script> that must not survive</p>'),
    ).not.toContain('alert(1)')
  })
})
