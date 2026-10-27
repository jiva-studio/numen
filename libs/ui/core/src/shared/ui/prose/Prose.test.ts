/**
 * Prose that carries links.
 *
 * What a link means is the caller's: this says only that a link was pressed
 * and hands over what it points at, so nothing here has to know a scheme.
 */
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Prose from './Prose.vue'

const { renderSpy } = vi.hoisted(() => ({ renderSpy: vi.fn() }))

vi.mock('./render', async (importOriginal) => {
  const original = await importOriginal<typeof import('./render')>()
  return {
    ...original,
    render: (...args: Parameters<typeof original.render>) => {
      renderSpy(args[0])
      return original.render(...args)
    },
  }
})

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

describe('a piece the caller names as a link', () => {
  const THERMO = {
    raw: '[[Thermodynamics]]',
    href: 'x://Thermodynamics',
    text: 'Thermodynamics',
  }
  const NOWHERE = { raw: '[[Nowhere]]', href: 'x://Nowhere', text: 'Nowhere' }

  it('is a link, pointing at the href it was named with', async () => {
    const wrapper = mount(Prose, {
      props: { text: 'It sits under [[Thermodynamics]].', links: [THERMO] },
    })

    await wrapper.find('a').trigger('click')

    expect(wrapper.find('a').text()).toBe('Thermodynamics')
    expect(wrapper.emitted('follow')?.[0]?.[0]).toBe('x://Thermodynamics')
  })

  it('is not followed by the browser', async () => {
    const wrapper = mount(Prose, {
      props: { text: 'It sits under [[Thermodynamics]].', links: [THERMO] },
    })

    await wrapper.find('a').trigger('click')

    const press = wrapper.emitted('follow')?.[0]?.[1] as MouseEvent
    expect(press.defaultPrevented).toBe(true)
  })

  it('draws plain text when no links are given', () => {
    expect(prose('It sits under [[Thermodynamics]].').find('a').exists()).toBe(false)
  })

  it('is drawn as reaching nothing where its href is unresolved', () => {
    const wrapper = mount(Prose, {
      props: {
        text: 'Under [[Nowhere]] and [[Thermodynamics]].',
        links: [NOWHERE, THERMO],
        unresolved: ['x://Nowhere'],
      },
    })

    const [nowhere, somewhere] = wrapper.findAll('a')
    expect(nowhere?.attributes('data-reaches')).toBe('nothing')
    expect(somewhere?.attributes('data-reaches')).toBeUndefined()
  })

  it('is not a link inside a code fence', () => {
    const wrapper = mount(Prose, {
      props: { text: '```\n[[Thermodynamics]]\n```', links: [THERMO] },
    })
    expect(wrapper.findAll('a')).toHaveLength(0)
  })

  it('is drawn once the caller names it', async () => {
    const wrapper = prose('See [[Thermodynamics]].')
    expect(wrapper.find('a').exists()).toBe(false)

    await wrapper.setProps({ links: [THERMO] })

    expect(wrapper.find('a').attributes('href')).toBe('x://Thermodynamics')
  })
})

describe('prose pressed anywhere else', () => {
  it('says nothing about a link', async () => {
    const wrapper = prose('Nothing here points anywhere.')

    await wrapper.find('p').trigger('click')

    expect(wrapper.emitted('follow')).toBeUndefined()
  })
})

describe('prose handed the same addresses again', () => {
  beforeEach(() => renderSpy.mockClear())

  it('is not drawn again while the unresolved addresses have the same members', async () => {
    const wrapper = mount(Prose, {
      props: {
        text: 'See [nowhere](https://example.com/nowhere).',
        unresolved: ['https://example.com/nowhere'],
      },
    })
    renderSpy.mockClear()

    await wrapper.setProps({ unresolved: ['https://example.com/nowhere'] })

    expect(renderSpy).not.toHaveBeenCalled()
  })

  it('is drawn again when the unresolved addresses change', async () => {
    const wrapper = mount(Prose, {
      props: { text: 'See [nowhere](https://example.com/nowhere).', unresolved: [] },
    })
    expect(wrapper.find('a').attributes('data-reaches')).toBeUndefined()
    renderSpy.mockClear()

    await wrapper.setProps({ unresolved: ['https://example.com/nowhere'] })

    expect(renderSpy).toHaveBeenCalledTimes(1)
    expect(wrapper.find('a').attributes('data-reaches')).toBeDefined()
  })

  it('is not drawn again while the links are the same', async () => {
    const link = { raw: '[[A]]', href: 'x://A', text: 'A' }
    const wrapper = mount(Prose, { props: { text: 'See [[A]].', links: [link] } })
    renderSpy.mockClear()

    await wrapper.setProps({ links: [{ ...link }] })

    expect(renderSpy).not.toHaveBeenCalled()
  })
})
