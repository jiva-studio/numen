/**
 * What an island card draws: its title in a header over the body, and the header's own slot only where one is given.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import IslandCard from './IslandCard.vue'

describe('the island card', () => {
  it('names itself in a header above the body', () => {
    const card = mount(IslandCard, {
      props: { title: 'Below a node' },
      slots: { default: '<ul><li>row</li></ul>' },
    })

    expect(card.get('header h3').text()).toBe('Below a node')
    expect(card.get('.island-card__body').text()).toBe('row')
    expect(card.find('header').element.nextElementSibling).toBe(
      card.get('.island-card__body').element,
    )
  })

  it('draws the aside of the header only where one is given', () => {
    const plain = mount(IslandCard, { props: { title: 't' } })
    const aside = mount(IslandCard, {
      props: { title: 't' },
      slots: { aside: '<button>Reset</button>' },
    })

    expect(plain.find('header button').exists()).toBe(false)
    expect(aside.get('header button').text()).toBe('Reset')
  })

  it('marks the body compact only when isCompact is set', () => {
    const roomy = mount(IslandCard, { props: { title: 't' } })
    const compact = mount(IslandCard, { props: { title: 't', isCompact: true } })

    expect(roomy.get('.island-card__body').classes()).not.toContain('island-card__body--compact')
    expect(compact.get('.island-card__body').classes()).toContain('island-card__body--compact')
  })
})
