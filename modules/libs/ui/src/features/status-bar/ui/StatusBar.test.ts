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
      { id: '1', label: 'Indexing Book 1', group: 'Books', progress: 50, state: 'running' },
      { id: '2', label: 'Indexing Book 2', group: 'Books', progress: 60, state: 'running' },
      { id: '3', label: 'Audio Transcribe', group: 'Audio', progress: 70, state: 'running' },
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
    expect(wrapper.text()).toContain('Audio Transcribe')
    expect(wrapper.text()).toContain('Indexing Book 1')

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
        error: 'Network error',
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

  const openBar = async () => {
    const wrapper = mount(StatusBar, {
      props: { tasks: [{ id: '1', label: 'Indexing', group: 'Books', state: 'running' }] },
      attachTo: document.body,
    })
    await wrapper.find('.status-bar__trigger').trigger('click')
    return wrapper
  }

  it('closes the popover on a press outside the bar', async () => {
    const wrapper = await openBar()

    document.body.dispatchEvent(new Event('pointerdown', { bubbles: true }))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.status-bar__popover-container').exists()).toBe(false)
  })

  it('keeps the popover open on a press inside the bar', async () => {
    const wrapper = await openBar()

    wrapper
      .find('.status-bar__popover-container')
      .element.dispatchEvent(new Event('pointerdown', { bubbles: true }))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.status-bar__popover-container').exists()).toBe(true)
  })

  it('closes the popover on Escape', async () => {
    const wrapper = await openBar()

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.status-bar__popover-container').exists()).toBe(false)
  })

  it('draws the labels it is given', () => {
    const wrapper = mount(StatusBar, {
      props: {
        tasks: [
          { id: '1', label: 'A', group: 'G', state: 'running' },
          { id: '2', label: 'B', group: 'G', state: 'failed' },
        ],
        activeLabel: 'в работе',
        failedLabel: 'с ошибкой',
      },
    })

    expect(wrapper.text()).toContain('1 в работе')
    expect(wrapper.text()).toContain('1 с ошибкой')
  })

  it('renders trailing slot content when provided', () => {
    const wrapper = mount(StatusBar, {
      props: {
        tasks: [],
      },
      slots: {
        trailing: '<div class="custom-trailing">Update available</div>',
      },
    })

    const trailing = wrapper.find('.status-bar__trailing')
    expect(trailing.exists()).toBe(true)
    expect(trailing.text()).toBe('Update available')
  })
})
