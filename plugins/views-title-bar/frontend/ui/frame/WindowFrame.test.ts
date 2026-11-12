/** The parts of the window frame the bar draws. */
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import WindowControls from './WindowControls.vue'
import WindowDragArea from './WindowDragArea.vue'
import MacButtonsInset from './MacButtonsInset.vue'
import { TITLE_BAR_WORDS } from '../../lib/words'

const mountControls = (props: Record<string, unknown> = {}) =>
  mount(WindowControls, { props: { look: 'windows', words: TITLE_BAR_WORDS, ...props } })

describe('WindowControls', () => {
  it('emits the press of each button', async () => {
    const wrapper = mountControls()

    await wrapper.find('[data-control="minimize"]').trigger('click')
    await wrapper.find('[data-control="toggle-maximize"]').trigger('click')
    await wrapper.find('[data-control="close"]').trigger('click')

    expect(wrapper.emitted('minimize-window')).toHaveLength(1)
    expect(wrapper.emitted('toggle-maximize-window')).toHaveLength(1)
    expect(wrapper.emitted('close-window')).toHaveLength(1)
  })

  it('names the maximize button Restore while the window is maximized', () => {
    const wrapper = mountControls({ isMaximized: true })

    expect(wrapper.find('[data-control="toggle-maximize"]').attributes('title')).toBe('Restore')
  })
})

describe('WindowDragArea', () => {
  it('holds its contents', () => {
    const wrapper = mount(WindowDragArea, { slots: { default: '<span data-held>Orchard</span>' } })

    expect(wrapper.find('[data-window-drag] [data-held]').exists()).toBe(true)
  })
})

describe('MacButtonsInset', () => {
  it('keeps the room of the system buttons', () => {
    expect(mount(MacButtonsInset).find('[data-mac-buttons-inset]').exists()).toBe(true)
  })

  it('keeps no room in full screen, where macOS shows no buttons', () => {
    const wrapper = mount(MacButtonsInset, { props: { isFullscreen: true } })

    expect(wrapper.find('[data-mac-buttons-inset]').exists()).toBe(false)
  })
})
