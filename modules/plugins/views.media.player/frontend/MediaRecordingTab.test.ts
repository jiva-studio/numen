import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MediaRecordingTab from './MediaRecordingTab.vue'

describe('MediaRecordingTab', () => {
  it('renders recording tab with idle state and start button', () => {
    const wrapper = mount(MediaRecordingTab, {
      props: {
        title: 'Voice Memo',
      },
    })

    expect(wrapper.text()).toContain('Voice Memo')
    expect(wrapper.text()).toContain('IDLE')
    expect(wrapper.find('.media-recorder__btn--record').exists()).toBe(true)
  })

  it('transitions state when recording starts and stops', async () => {
    const wrapper = mount(MediaRecordingTab)

    const recordBtn = wrapper.find('.media-recorder__btn--record')
    await recordBtn.trigger('click')

    expect(wrapper.emitted('record-start')).toBeTruthy()
    expect(wrapper.text()).toContain('RECORDING')

    const stopBtn = wrapper.find('.media-recorder__btn--stop')
    expect(stopBtn.exists()).toBe(true)
    await stopBtn.trigger('click')

    expect(wrapper.emitted('record-stop')).toBeTruthy()
    expect(wrapper.text()).toContain('STOPPED')
  })

  it('resets back to idle state on new recording request', async () => {
    const wrapper = mount(MediaRecordingTab)

    await wrapper.find('.media-recorder__btn--record').trigger('click')
    await wrapper.find('.media-recorder__btn--stop').trigger('click')

    expect(wrapper.text()).toContain('STOPPED')

    const newBtn = wrapper.find('.media-recorder__btn--new')
    expect(newBtn.exists()).toBe(true)
    await newBtn.trigger('click')

    expect(wrapper.text()).toContain('IDLE')
  })
})
