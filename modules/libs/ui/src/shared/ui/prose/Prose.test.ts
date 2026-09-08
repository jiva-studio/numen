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

describe('a wikilink in the prose', () => {
  it('is a link, pointing at the address the brackets hold', async () => {
    const wrapper = prose('It sits under [[Thermodynamics]].')

    await wrapper.find('a').trigger('click')

    expect(wrapper.find('a').text()).toBe('Thermodynamics')
    expect(wrapper.emitted('follow')?.[0]?.[0]).toBe('name://Thermodynamics')
  })

  it('is read in the sentence by its alias', () => {
    const wrapper = prose('It argues [[Entropy as disorder|the other way]].')

    expect(wrapper.find('a').text()).toBe('the other way')
    expect(wrapper.find('a').attributes('href')).toBe('name://Entropy as disorder')
  })

  it('is not followed by the browser, which has nowhere to take a note', async () => {
    const wrapper = prose('It sits under [[Thermodynamics]].')

    await wrapper.find('a').trigger('click')

    const press = wrapper.emitted('follow')?.[0]?.[1] as MouseEvent
    expect(press.defaultPrevented).toBe(true)
  })

  it('is drawn as reaching nothing where nothing answers to it', () => {
    const wrapper = mount(Prose, {
      props: { text: 'Under [[Nowhere]] and [[Thermodynamics]].', unresolved: ['name://Nowhere'] },
    })

    const [nowhere, somewhere] = wrapper.findAll('a')
    expect(nowhere?.attributes('data-reaches')).toBe('nothing')
    expect(somewhere?.attributes('data-reaches')).toBeUndefined()
  })

  it('is an example of a link inside a code fence, and not one', () => {
    expect(prose('```\n[[Thermodynamics]]\n```').findAll('a')).toHaveLength(0)
  })
})

describe('prose pressed anywhere else', () => {
  it('says nothing about a link', async () => {
    const wrapper = prose('Nothing here points anywhere.')

    await wrapper.find('p').trigger('click')

    expect(wrapper.emitted('follow')).toBeUndefined()
  })
})
