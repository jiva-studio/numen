import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { APPEAR_DELAY_MS } from '../lib/chipDisplay'
import { createTask } from '../testing/tasks'
import TasksStatus from './TasksStatus.vue'

beforeEach(() => vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'performance'] }))
afterEach(() => vi.useRealTimers())

describe('TasksStatus', () => {
  it('shows Idle until a running task has lasted the appearance delay', async () => {
    const wrapper = mount(TasksStatus, { props: { tasks: [] } })

    await wrapper.setProps({ tasks: [createTask({ id: 'a', state: 'running' })] })
    expect(wrapper.find('button').text()).toBe('Idle')

    vi.advanceTimersByTime(APPEAR_DELAY_MS)
    await flushPromises()
    expect(wrapper.find('button').text()).toBe('0 of 1')
    wrapper.unmount()
  })

  it('counts the failure at once', async () => {
    const wrapper = mount(TasksStatus, { props: { tasks: [] } })

    await wrapper.setProps({
      tasks: [createTask({ id: 'a', state: 'failed', error: 'disk full' })],
    })

    const trigger = wrapper.find('button')
    expect(trigger.classes()).toContain('bar-chip--alarm')
    expect(trigger.attributes('aria-label')).toBe('1 failed')
    wrapper.unmount()
  })
})
