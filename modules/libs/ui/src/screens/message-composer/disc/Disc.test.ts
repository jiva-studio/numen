import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Disc from './Disc.vue'

const disc = (props: Record<string, unknown> = {}, slots: Record<string, string> = {}) =>
  mount(Disc, {
    props: { action: 'send', disabled: false, sendLabel: 'Send', stopLabel: 'Stop', ...props },
    slots,
  })

describe('the disc', () => {
  it('is called what it does', () => {
    expect(disc().get('button').attributes('aria-label')).toBe('Send')
    expect(disc({ action: 'stop' }).get('button').attributes('aria-label')).toBe('Stop')
  })

  it('says it was pressed', async () => {
    const wrapper = disc()
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('press')).toStrictEqual([[]])
  })

  it('says nothing while it is barred', () => {
    expect(disc({ disabled: true }).get('button').attributes('disabled')).toBeDefined()
  })

  it('stands one disc either way, and changes the glyph on it', () => {
    const glyph = (action: string) => {
      const wrapper = disc({ action })
      expect(wrapper.findAll('button')).toHaveLength(1)
      return wrapper.get('button svg').html()
    }
    expect(glyph('stop')).not.toBe(glyph('send'))
  })

  it('takes a glyph of its own while it sends, and keeps its own while it stops', () => {
    const given = { glyph: '<i class="mine" />' }
    expect(disc({}, given).find('.mine').exists()).toBe(true)
    expect(disc({ action: 'stop' }, given).find('.mine').exists()).toBe(false)
  })
})
