import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Composer from './Composer.vue'

const composer = (props: Record<string, unknown> = {}) =>
  mount(Composer, { props: { modelValue: '', ...props } })

const enter = { key: 'Enter' }

describe('sending', () => {
  it('says what was written, with the whitespace around it gone', async () => {
    const wrapper = composer({ modelValue: '  hello  ' })
    await wrapper.get('textarea').trigger('keydown', enter)
    expect(wrapper.emitted('submit')).toStrictEqual([['hello']])
  })

  it('sends when the button is pressed', async () => {
    const wrapper = composer({ modelValue: 'hello' })
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('submit')).toStrictEqual([['hello']])
  })

  it('does not clear the field, which belongs to whoever answers', async () => {
    const wrapper = composer({ modelValue: 'hello' })
    await wrapper.get('textarea').trigger('keydown', enter)
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })
})

describe('what is not sent', () => {
  it('sends nothing when nothing was written', async () => {
    const wrapper = composer({ modelValue: '' })
    await wrapper.get('textarea').trigger('keydown', enter)
    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('sends nothing when only whitespace was written', async () => {
    const wrapper = composer({ modelValue: '   \n  ' })
    await wrapper.get('textarea').trigger('keydown', enter)
    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('sends nothing on Shift+Enter', async () => {
    const wrapper = composer({ modelValue: 'hello' })
    await wrapper.get('textarea').trigger('keydown', { key: 'Enter', shiftKey: true })
    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('sends nothing while an input method is composing', async () => {
    const wrapper = composer({ modelValue: 'こんにちは' })
    await wrapper.get('textarea').trigger('keydown', { key: 'Enter', isComposing: true })
    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('sends nothing when the key is reported as composing', async () => {
    const wrapper = composer({ modelValue: 'こんにちは' })
    await wrapper.get('textarea').trigger('keydown', { key: 'Enter', keyCode: 229 })
    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('sends nothing while turned off', async () => {
    const wrapper = composer({ modelValue: 'hello', disabled: true })
    await wrapper.get('textarea').trigger('keydown', enter)
    expect(wrapper.emitted('submit')).toBeUndefined()
  })
})

describe('while an answer is being written', () => {
  it('gives the button’s place to the dots', () => {
    const wrapper = composer({ modelValue: 'hello', working: true })
    expect(wrapper.find('button').exists()).toBe(false)
    expect(wrapper.find('.dots').exists()).toBe(true)
  })

  it('sends nothing on Enter', async () => {
    const wrapper = composer({ modelValue: 'hello', working: true })
    await wrapper.get('textarea').trigger('keydown', enter)
    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('stands the dots on the disc the button stood on', () => {
    const disc = (working: boolean) => {
      const wrapper = composer({ modelValue: 'hello', working })
      const element = working ? wrapper.get('.dots').element.parentElement : wrapper.get('button').element
      return [...(element?.classList ?? [])].filter((name) => name.startsWith('size-') || name.startsWith('rounded-'))
    }
    expect(disc(true)).toStrictEqual(disc(false))
  })

  it('shows the button again once the answer has arrived', async () => {
    const wrapper = composer({ modelValue: 'hello', working: true })
    await wrapper.setProps({ working: false })
    expect(wrapper.find('button').exists()).toBe(true)
    expect(wrapper.find('.dots').exists()).toBe(false)
  })
})

describe('what cannot be reached', () => {
  it('disables the button with nothing to send', () => {
    expect(composer({ modelValue: '' }).get('button').attributes('disabled')).toBeDefined()
  })

  it('disables the field when the composer is turned off', () => {
    const wrapper = composer({ modelValue: 'hello', disabled: true })
    expect(wrapper.get('textarea').attributes('disabled')).toBeDefined()
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
  })
})
