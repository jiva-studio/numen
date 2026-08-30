// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'

import Card from './Card.vue'
import { FAR, STILL } from './swiping'

const shows = (front: string, said: { back?: string; shown?: boolean; asking?: boolean } = {}) =>
  mount(Card, {
    props: {
      front,
      back: said.back ?? '<p>the back</p>',
      shown: said.shown ?? false,
      asking: said.asking ?? false,
    },
  })

/**
 * A hand going down on the card, across by so much, and up again. A card takes
 * the pointer while a hand is on it, and jsdom has no capture to give it.
 */
const taken = async (face: ReturnType<typeof shows>, across: number) => {
  const card = face.find('.card').element as HTMLElement
  card.setPointerCapture = () => {}
  card.dispatchEvent(new MouseEvent('pointerdown', { clientX: 100 }))
  card.dispatchEvent(new MouseEvent('pointermove', { clientX: 100 + across }))
  card.dispatchEvent(new MouseEvent('pointerup', { clientX: 100 + across }))
  await nextTick()
}

describe('a card', () => {
  it('draws the front, and the back once the answer is showing', () => {
    const face = shows('<p>Leaf mould</p>', { back: '<p>Compost</p>' })
    expect(face.text()).toContain('Leaf mould')
    expect(face.text()).not.toContain('Compost')

    const over = shows('<p>Leaf mould</p>', { back: '<p>Compost</p>', shown: true })
    expect(over.text()).toContain('Compost')
  })

  it('draws what a card is written with', () => {
    const face = shows('<p><strong>bold</strong> and <em>italic</em></p><ul><li>a list</li></ul>')
    expect(face.find('strong').exists()).toBe(true)
    expect(face.find('em').exists()).toBe(true)
    expect(face.find('li').exists()).toBe(true)
  })

  // A deck may have come from another person, and what they wrote is not this
  // window's to run, style or submit.
  it('draws nothing a card may not be drawn with', () => {
    const face = shows(
      '<p>before</p>' +
        '<script>window.ran = true</script>' +
        '<style>body { display: none }</style>' +
        '<form action="https://elsewhere.example"><input name="what" /></form>' +
        '<iframe src="https://elsewhere.example"></iframe>' +
        '<p onclick="window.ran = true">after</p>',
    )

    expect(face.html()).not.toContain('<script')
    expect(face.html()).not.toContain('<style')
    expect(face.html()).not.toContain('<form')
    expect(face.html()).not.toContain('<iframe')
    expect(face.html()).not.toContain('onclick')
    // What the card actually says is still there.
    expect(face.text()).toContain('before')
    expect(face.text()).toContain('after')
  })

  it('turns over when it is pressed, and only while the answer is hidden', async () => {
    const face = shows('<p>Leaf mould</p>')
    await face.find('.card').trigger('click')
    expect(face.emitted('show')).toHaveLength(1)

    const over = shows('<p>Leaf mould</p>', { shown: true })
    await over.find('.card').trigger('click')
    expect(over.emitted('show')).toBeUndefined()
  })

  // A card that can be asked about before it is turned is a way not to recall
  // it.
  it('is not dragged while the answer is hidden, however far it is taken', async () => {
    const face = shows('<p>Leaf mould</p>')
    await taken(face, -FAR * 4)
    expect(face.emitted('ask')).toBeUndefined()
    expect(face.emitted('dragging')).toBeUndefined()
  })

  it('brings the panel in when it is taken far to the left', async () => {
    const face = shows('<p>Leaf mould</p>', { shown: true })
    await taken(face, -FAR)
    expect(face.emitted('ask')).toHaveLength(1)
    expect(face.emitted('shut')).toBeUndefined()
  })

  it('sends the panel away when it is taken far back to the right', async () => {
    const face = shows('<p>Leaf mould</p>', { shown: true, asking: true })
    await taken(face, FAR)
    expect(face.emitted('shut')).toHaveLength(1)
    expect(face.emitted('ask')).toBeUndefined()
  })

  it('asks for nothing when it is barely taken anywhere', async () => {
    const face = shows('<p>Leaf mould</p>', { shown: true })
    await taken(face, -STILL)
    expect(face.emitted('ask')).toBeUndefined()
    expect(face.emitted('shut')).toBeUndefined()
  })

  it('asks for nothing when it is taken less far than the panel wants', async () => {
    const face = shows('<p>Leaf mould</p>', { shown: true })
    await taken(face, -(FAR - 1))
    expect(face.emitted('ask')).toBeUndefined()
    expect(face.emitted('shut')).toBeUndefined()
  })

  // This window has one page and no way back to it.
  it('does not follow a link a card carries', async () => {
    const face = shows('<p><a href="https://elsewhere.example">elsewhere</a></p>')
    const link = face.find('a')
    expect(link.exists()).toBe(true)

    await link.trigger('click')
    expect(face.emitted('show')).toBeUndefined()
  })
})
