import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import AudioPlayerTab from './AudioPlayerTab.vue'

describe('AudioPlayerTab', () => {
  it('renders audio title, controls, and waveform', () => {
    const wrapper = mount(AudioPlayerTab, {
      props: {
        path: 'talks/lecture.mp3',
        title: 'Lecture 1',
      },
    })

    expect(wrapper.text()).toContain('Lecture 1')
    expect(wrapper.find('.audio-player__waveform').exists()).toBe(true)
    expect(wrapper.findAll('.audio-player__waveform-bar').length).toBe(64)
    expect(wrapper.find('.audio-player__btn--play').exists()).toBe(true)
  })

  it('renders playback speed options and handles rate changes', async () => {
    const wrapper = mount(AudioPlayerTab, {
      props: {
        path: 'talks/lecture.mp3',
      },
    })

    const speedButtons = wrapper.findAll('.audio-player__speed-btn')
    expect(speedButtons.length).toBe(6)

    const button15 = speedButtons.find((btn) => btn.text() === '1.5x')
    expect(button15).toBeDefined()
    await button15!.trigger('click')

    expect(wrapper.emitted('rateChange')).toBeTruthy()
    expect(wrapper.emitted('rateChange')![0]).toEqual([1.5])
  })

  it('navigates transcript timestamps when cues are clicked', async () => {
    const cues = [
      { text: 'Opening remarks', from: 0, to: 3000 },
      { text: 'Core argument', from: 45000, to: 50000 },
    ]

    const wrapper = mount(AudioPlayerTab, {
      props: {
        path: 'talks/lecture.mp3',
        state: { cues, runs: 60000 },
      },
    })

    const cueElements = wrapper.findAll('.audio-player__cue')
    expect(cueElements.length).toBe(2)
    expect(wrapper.text()).toContain('Opening remarks')
    expect(wrapper.text()).toContain('Core argument')

    const secondTimestamp = cueElements[1]!.find('.audio-player__cue-timestamp')
    await secondTimestamp.trigger('click')

    expect(wrapper.emitted('seek')).toBeTruthy()
    expect(wrapper.emitted('seek')![0]).toEqual([45000])
  })

  it('seeks when clicking on waveform visualization', async () => {
    const wrapper = mount(AudioPlayerTab, {
      props: {
        path: 'talks/lecture.mp3',
        state: { runs: 100000 },
      },
    })

    const waveform = wrapper.find('.audio-player__waveform')
    expect(waveform.exists()).toBe(true)
    await waveform.trigger('click')

    expect(wrapper.emitted('seek')).toBeTruthy()
  })
})
