// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { nextTick } from 'vue'

import Card from './Card.vue'

const mountCard = (front: string, said: { back?: string; shown?: boolean } = {}) =>
  mount(Card, {
    props: { front, back: said.back ?? '<p>the back</p>', shown: said.shown ?? false },
  })

/** A hand going down on the card, across by so much, and up again. */
const taken = async (face: ReturnType<typeof mountCard>, across: number) => {
  const card = face.find('.card').element as HTMLElement
  card.dispatchEvent(new MouseEvent('pointerdown', { clientX: 100 }))
  card.dispatchEvent(new MouseEvent('click', { clientX: 100 + across }))
  await nextTick()
}

describe('a card', () => {
  it('draws the front, and the back once the answer is showing', () => {
    const face = mountCard('<p>Leaf mould</p>', { back: '<p>Compost</p>' })
    expect(face.text()).toContain('Leaf mould')
    expect(face.text()).not.toContain('Compost')

    const over = mountCard('<p>Leaf mould</p>', { back: '<p>Compost</p>', shown: true })
    expect(over.text()).toContain('Compost')
  })

  // A card is HTML, so the marks of another format are the characters somebody
  // typed and nothing more.
  it('draws the marks a card carries as the text they are', () => {
    const face = mountCard('**bold** and *italic*', { back: '- a list', shown: true })
    expect(face.find('strong').exists()).toBe(false)
    expect(face.find('em').exists()).toBe(false)
    expect(face.find('li').exists()).toBe(false)
    expect(face.text()).toContain('**bold** and *italic*')
    expect(face.text()).toContain('- a list')
  })

  it('draws what a card is written with', () => {
    const face = mountCard('<p><strong>bold</strong> and <em>italic</em></p><ul><li>a list</li></ul>')
    expect(face.find('strong').exists()).toBe(true)
    expect(face.find('em').exists()).toBe(true)
    expect(face.find('li').exists()).toBe(true)
  })

  // A deck may have come from another person, and what they wrote is not this
  // window's to run, style or submit.
  it('draws nothing a card may not be drawn with', () => {
    const face = mountCard(
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
    const face = mountCard('<p>Leaf mould</p>')
    await face.find('.card').trigger('click')
    expect(face.emitted('show')).toHaveLength(1)

    const over = mountCard('<p>Leaf mould</p>', { shown: true })
    await over.find('.card').trigger('click')
    expect(over.emitted('show')).toBeUndefined()
  })

  // A hand that took the card across was moving the panel into view, and the
  // card is not turned over on the way.
  it('is not turned over by a hand that took it across', async () => {
    const face = mountCard('<p>Leaf mould</p>')
    await taken(face, -120)
    expect(face.emitted('show')).toBeUndefined()
  })

  it('is turned over by a hand that went nowhere', async () => {
    const face = mountCard('<p>Leaf mould</p>')
    await taken(face, 0)
    expect(face.emitted('show')).toHaveLength(1)
  })

  // This window has one page and no way back to it.
  it('does not follow a link a card carries', async () => {
    const face = mountCard('<p><a href="https://elsewhere.example">elsewhere</a></p>')
    const link = face.find('a')
    expect(link.exists()).toBe(true)

    await link.trigger('click')
    expect(face.emitted('show')).toBeUndefined()
    expect(face.emitted('read')).toBeUndefined()
  })

  // A name carrying no scheme points inside the vault, and that is what stands
  // in the reading beside the card.
  it('opens the reading on a note a link inside it names', async () => {
    const face = mountCard('<p><a href="notes/Leaf%20mould.md">leaf mould</a></p>')

    await face.find('a').trigger('click')
    expect(face.emitted('show')).toBeUndefined()
    expect(face.emitted('read')).toEqual([['notes/Leaf mould.md']])
  })

  // A card comes from whoever wrote it, and a name escaped wrongly is still a
  // press this window has to survive.
  it('takes a badly escaped name as the characters it already is', async () => {
    const face = mountCard('<p><a href="notes/50%.md">fifty</a></p>')

    await face.find('a').trigger('click')
    expect(face.emitted('read')).toEqual([['notes/50%.md']])
  })
})
