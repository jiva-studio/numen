import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import StatusBar from './StatusBar.vue'
import type { StatusBarTask } from '../model/types'

describe('StatusBar', () => {
  it('renders idle state with idle label when no tasks are present', () => {
    const wrapper = mount(StatusBar, {
      props: {
        tasks: [],
        idleLabel: 'Ready',
      },
    })

    expect(wrapper.text()).toContain('Ready')
    expect(wrapper.find('.status-bar__progress-track').exists()).toBe(false)
  })

  it('renders active aggregate information when tasks are running', () => {
    const tasks: StatusBarTask[] = [
      { id: '1', label: 'Indexing Book', group: 'Books', progress: 50, state: 'running' },
      {
        id: '2',
        label: 'Audio Transcribe',
        group: 'Audio',
        progress: 70,
        state: 'running',
        etaSeconds: 90,
      },
    ]

    const wrapper = mount(StatusBar, {
      props: {
        tasks,
      },
    })

    expect(wrapper.text()).toContain('2 active')
  })

  it('toggles task popover when status bar button is clicked', async () => {
    const tasks: StatusBarTask[] = [
      { id: '1', label: 'Indexing Book', group: 'Books', progress: 50, state: 'running' },
      { id: '2', label: 'Audio Transcribe', group: 'Audio', progress: 70, state: 'running' },
    ]

    const wrapper = mount(StatusBar, {
      props: {
        tasks,
      },
      attachTo: document.body,
    })

    expect(wrapper.find('.status-bar__popover-container').exists()).toBe(false)

    await wrapper.find('.status-bar__trigger').trigger('click')
    expect(wrapper.find('.status-bar__popover-container').exists()).toBe(true)
    expect(wrapper.text()).toContain('Books')
    expect(wrapper.text()).toContain('Audio')
    expect(wrapper.text()).toContain('Indexing Book')

    await wrapper.find('.status-bar__trigger').trigger('click')
    expect(wrapper.find('.status-bar__popover-container').exists()).toBe(false)
  })

  it('forwards cancel event when cancel button is clicked', async () => {
    const tasks: StatusBarTask[] = [
      {
        id: 'task-abc',
        label: 'Long Process',
        group: 'General',
        state: 'running',
        canCancel: true,
      },
    ]

    const wrapper = mount(StatusBar, {
      props: {
        tasks,
      },
      attachTo: document.body,
    })

    await wrapper.find('.status-bar__trigger').trigger('click')
    const cancelBtn = wrapper.find('button[aria-label="Cancel task"]')
    expect(cancelBtn.exists()).toBe(true)

    await cancelBtn.trigger('click')
    expect(wrapper.emitted('cancel')).toBeTruthy()
    expect(wrapper.emitted('cancel')?.[0]).toEqual(['task-abc'])
  })

  it('displays failed count badge and forwards retry/dismiss events', async () => {
    const tasks: StatusBarTask[] = [
      {
        id: 'failed-1',
        label: 'Failed Task',
        group: 'Import',
        state: 'failed',
        errorReason: 'Network error',
      },
      {
        id: 'completed-1',
        label: 'Done Task',
        group: 'Import',
        state: 'completed',
      },
    ]

    const wrapper = mount(StatusBar, {
      props: {
        tasks,
      },
      attachTo: document.body,
    })

    expect(wrapper.text()).toContain('1 failed')

    await wrapper.find('.status-bar__trigger').trigger('click')
    expect(wrapper.text()).toContain('Network error')

    const retryBtn = wrapper.find('button[aria-label="Retry task"]')
    await retryBtn.trigger('click')
    expect(wrapper.emitted('retry')?.[0]).toEqual(['failed-1'])

    const dismissBtn = wrapper.find('button[aria-label="Dismiss"]')
    await dismissBtn.trigger('click')
    expect(wrapper.emitted('dismiss')?.[0]).toEqual(['completed-1'])
  })
})
