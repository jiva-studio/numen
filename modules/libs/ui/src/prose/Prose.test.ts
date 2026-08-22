/**
 * Prose that carries links.
 *
 * What a link means is the caller's: this says only that a link was pressed
 * and hands over what it points at, so nothing here has to know a scheme.
 */
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Prose from './Prose.vue'

const prose = (text: string) => mount(Prose, { props: { text } })

describe('a link that was pressed', () => {
  it('says what it points at', async () => {
    const wrapper = prose('Read [the passage](numen:book.pdf?start=1&length=2) of it.')

    await wrapper.find('a').trigger('click')

    const followed = wrapper.emitted('follow')
    expect(followed).toHaveLength(1)
    expect(followed?.[0]?.[0]).toBe('numen:book.pdf?start=1&length=2')
  })

  it('is not followed here, whatever it points at', async () => {
    const wrapper = prose('Read [the site](https://example.com) about it.')

    await wrapper.find('a').trigger('click')

    const press = wrapper.emitted('follow')?.[0]?.[1] as MouseEvent
    expect(press.defaultPrevented).toBe(false)
  })
})

describe('prose pressed anywhere else', () => {
  it('says nothing about a link', async () => {
    const wrapper = prose('Nothing here points anywhere.')

    await wrapper.find('p').trigger('click')

    expect(wrapper.emitted('follow')).toBeUndefined()
  })
})
