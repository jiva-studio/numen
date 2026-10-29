// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'

import Card from './Card.vue'

const mountCard = (front: string, overrides: { back?: string; isShown?: boolean } = {}) =>
  mount(Card, {
    props: {
      front,
      back: overrides.back ?? '<p>the back</p>',
      isShown: overrides.isShown ?? false,
    },
  })

/** A pointer drag on the card by the given distance, followed by release. */
const dragCard = async (reviewCard: ReturnType<typeof mountCard>, distance: number) => {
  const card = reviewCard.find('.card').element as HTMLElement
  card.dispatchEvent(new MouseEvent('pointerdown', { clientX: 100 }))
  card.dispatchEvent(new MouseEvent('click', { clientX: 100 + distance }))
  await nextTick()
}

describe('a card', () => {
  it('draws the front, and the back once the answer is showing', () => {
    const reviewCard = mountCard('<p>Leaf mould</p>', { back: '<p>Compost</p>' })
    expect(reviewCard.text()).toContain('Leaf mould')
    expect(reviewCard.text()).not.toContain('Compost')

    const wrapper = mountCard('<p>Leaf mould</p>', { back: '<p>Compost</p>', isShown: true })
    expect(wrapper.text()).toContain('Compost')
  })

  // A card is HTML, so the marks of another format are plain characters and
  // nothing more.
  it('draws the marks a card carries as the text they are', () => {
    const reviewCard = mountCard('**bold** and *italic*', { back: '- a list', isShown: true })
    expect(reviewCard.find('strong').exists()).toBe(false)
    expect(reviewCard.find('em').exists()).toBe(false)
    expect(reviewCard.find('li').exists()).toBe(false)
    expect(reviewCard.text()).toContain('**bold** and *italic*')
    expect(reviewCard.text()).toContain('- a list')
  })

  it('draws what a card is written with', () => {
    const reviewCard = mountCard(
      '<p><strong>bold</strong> and <em>italic</em></p><ul><li>a list</li></ul>',
    )
    expect(reviewCard.find('strong').exists()).toBe(true)
    expect(reviewCard.find('em').exists()).toBe(true)
    expect(reviewCard.find('li').exists()).toBe(true)
  })

  // A deck may have come from another person, and what they wrote is not this
  // window's to run, style or submit.
  it('draws nothing a card may not be drawn with', () => {
    const reviewCard = mountCard(
      '<p>before</p>' +
        '<script>window.ran = true</script>' +
        '<style>body { display: none }</style>' +
        '<form action="https://elsewhere.example"><input name="what" /></form>' +
        '<iframe src="https://elsewhere.example"></iframe>' +
        '<p onclick="window.ran = true">after</p>',
    )

    expect(reviewCard.html()).not.toContain('<script')
    expect(reviewCard.html()).not.toContain('<style')
    expect(reviewCard.html()).not.toContain('<form')
    expect(reviewCard.html()).not.toContain('<iframe')
    expect(reviewCard.html()).not.toContain('onclick')
    // The card's actual text is still present.
    expect(reviewCard.text()).toContain('before')
    expect(reviewCard.text()).toContain('after')
  })

  it('turns over when it is pressed, and only while the answer is hidden', async () => {
    const reviewCard = mountCard('<p>Leaf mould</p>')
    await reviewCard.find('.card').trigger('click')
    expect(reviewCard.emitted('show')).toHaveLength(1)

    const wrapper = mountCard('<p>Leaf mould</p>', { isShown: true })
    await wrapper.find('.card').trigger('click')
    expect(wrapper.emitted('show')).toBeUndefined()
  })

  // card is not turned over on the way.
  it('does not flip card on swipe gesture across threshold', async () => {
    const reviewCard = mountCard('<p>Leaf mould</p>')
    await dragCard(reviewCard, -120)
    expect(reviewCard.emitted('show')).toBeUndefined()
  })

  it('is turned over by a hand that went nowhere', async () => {
    const reviewCard = mountCard('<p>Leaf mould</p>')
    await dragCard(reviewCard, 0)
    expect(reviewCard.emitted('show')).toHaveLength(1)
  })

  // This window has one page and no way back to it.
  it('does not follow a link a card carries', async () => {
    const reviewCard = mountCard('<p><a href="https://elsewhere.example">elsewhere</a></p>')
    const link = reviewCard.find('a')
    expect(link.exists()).toBe(true)

    await link.trigger('click')
    expect(reviewCard.emitted('show')).toBeUndefined()
    expect(reviewCard.emitted('read')).toBeUndefined()
  })

  // A name without a scheme points inside the vault, and that note is opened in
  // the reading panel beside the card.
  it('opens the reading on a note a link inside it names', async () => {
    const reviewCard = mountCard('<p><a href="notes/Leaf%20mould.md">leaf mould</a></p>')

    await reviewCard.find('a').trigger('click')
    expect(reviewCard.emitted('show')).toBeUndefined()
    expect(reviewCard.emitted('read')).toEqual([['notes/Leaf mould.md']])
  })

  // A card's text comes from its author, and a wrongly escaped name must not
  // break the window.
  it('takes a badly escaped name as the characters it already is', async () => {
    const reviewCard = mountCard('<p><a href="notes/50%.md">fifty</a></p>')

    await reviewCard.find('a').trigger('click')
    expect(reviewCard.emitted('read')).toEqual([['notes/50%.md']])
  })
})
