import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import MessageComposer from './MessageComposer.vue'

const composer = (props: Record<string, unknown> = {}) =>
  mount(MessageComposer, { props: { modelValue: '', ...props } })

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
  it('gives the disc to stopping, under the name it was given', async () => {
    const wrapper = composer({ modelValue: 'hello', working: true, stops: 'Give up' })
    const disc = wrapper.get('button')
    expect(disc.attributes('aria-label')).toBe('Give up')
    expect(disc.attributes('disabled')).toBeUndefined()

    await disc.trigger('click')
    expect(wrapper.emitted('stop')).toStrictEqual([[]])
    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('sends nothing on Enter', async () => {
    const wrapper = composer({ modelValue: 'hello', working: true })
    await wrapper.get('textarea').trigger('keydown', enter)
    expect(wrapper.emitted('submit')).toBeUndefined()
  })

  it('stops nothing on Enter', async () => {
    const wrapper = composer({ modelValue: 'hello', working: true })
    await wrapper.get('textarea').trigger('keydown', enter)
    expect(wrapper.emitted('stop')).toBeUndefined()
  })

  it('stands one disc either way, and changes the glyph on it', () => {
    const glyph = (isWriting: boolean) => {
      const wrapper = composer({ modelValue: 'hello', working: isWriting })
      expect(wrapper.findAll('button')).toHaveLength(1)
      return wrapper.get('button svg').html()
    }
    expect(glyph(true)).not.toBe(glyph(false))
  })

  it('sends again once the answer has arrived', async () => {
    const wrapper = composer({ modelValue: 'hello', working: true, sends: 'Send it' })
    await wrapper.setProps({ working: false })
    expect(wrapper.get('button').attributes('aria-label')).toBe('Send it')

    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('submit')).toStrictEqual([['hello']])
    expect(wrapper.emitted('stop')).toBeUndefined()
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
