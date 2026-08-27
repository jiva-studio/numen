/**
 * What a face draws from the markdown it was handed, and what it emits.
 *
 * The negatives are here: the back is not drawn until the card is turned, and
 * a half with nothing in it draws no prose at all.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import Face from './Face.vue'

const FRONT = '# Llama\n\nWhat is it?'
const BACK = '**Height:** about 45"'

const mountFace = (props: Record<string, unknown> = {}) =>
  mount(Face, { attachTo: document.body, props: { front: FRONT, back: BACK, ...props } })

afterEach(() => {
  document.body.innerHTML = ''
})

describe('Face', () => {
  it('draws the front’s markdown as marks and not as text', () => {
    const held = mountFace()
    expect(held.get('[data-half="front"] h1').text()).toBe('Llama')
    expect(held.text()).not.toContain('#')
  })

  it('does not draw the back while the card is not turned', () => {
    const held = mountFace()
    expect(held.find('[data-half="back"]').exists()).toBe(false)
    expect(held.text()).not.toContain('45"')
  })

  it('draws the back once the card is turned', () => {
    const held = mountFace({ turned: true })
    expect(held.get('[data-half="back"]').text()).toContain('about 45"')
    expect(held.get('[data-half="back"] strong').text()).toBe('Height:')
  })

  it('says the silence in place of a half with nothing in it', () => {
    const held = mountFace({ front: '   \n ', silence: 'Nothing' })
    expect(held.get('.face__silence').text()).toBe('Nothing')
    expect(held.find('[data-half="front"] .marks').exists()).toBe(false)
  })

  it('turns the card by a button and not by a word drawn as one', () => {
    const turn = mountFace().get('.face__turn')
    expect(turn.element.tagName).toBe('BUTTON')
    expect(turn.attributes('type')).toBe('button')
  })

  it('draws the tags a person wrote among the marks', () => {
    const held = mountFace({ front: 'a <u>marked</u> word' })
    expect(held.get('[data-half="front"] u').text()).toBe('marked')
  })

  it('draws no script a person wrote among the marks', () => {
    const held = mountFace({ front: '<script>alert(1)</script>after' })
    expect(held.find('[data-half="front"] script').exists()).toBe(false)
    expect(held.html()).not.toContain('alert(1)')
  })

  it('draws no handler a person wrote among the marks', () => {
    const held = mountFace({ front: '<img src="x" onerror="alert(1)">' })
    expect(held.get('[data-half="front"] img').attributes('onerror')).toBeUndefined()
  })

  it('emits the side turning would come to', async () => {
    const held = mountFace()
    await held.get('.face__turn').trigger('click')
    expect(held.emitted('turn')).toEqual([[true]])
  })

  it('emits the other side from a card already turned', async () => {
    const held = mountFace({ turned: true })
    await held.get('.face__turn').trigger('click')
    expect(held.emitted('turn')).toEqual([[false]])
  })

  it('turns nothing itself: what is drawn follows the prop alone', async () => {
    const held = mountFace()
    await held.get('.face__turn').trigger('click')
    expect(held.find('[data-half="back"]').exists()).toBe(false)
  })

  it('emits what a link in the card points at', async () => {
    const held = mountFace({ front: '[a](there)' })
    await held.get('.marks a').trigger('click')
    expect(held.emitted('follow')?.[0]?.[0]).toBe('there')
  })

  it('emits nothing for a press that lands on no link', async () => {
    const held = mountFace({ front: 'plain words' })
    await held.get('.marks').trigger('click')
    expect(held.emitted('follow')).toBeUndefined()
  })

  it('is announced by the name it was given', () => {
    expect(mountFace({ name: 'Llama' }).get('article').attributes('aria-label')).toBe('Llama')
  })
})
