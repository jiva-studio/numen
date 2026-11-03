import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import TaskPopover from './TaskPopover.vue'
import type { StatusBarTask } from '../model/types'

function mountTasks(tasks: StatusBarTask[]) {
  return mount(TaskPopover, { props: { tasks } })
}

describe('TaskPopover', () => {
  it('lists all tasks as one flat list, newest first, without group headers', () => {
    const wrapper = mountTasks([
      { id: '1', label: 'Reader task', state: 'running' },
      { id: '2', label: 'Embedding vault', state: 'running' },
      { id: '3', label: 'Walking the notes', state: 'running' },
    ])

    const rows = wrapper.findAll('.task-item').map((row) => row.attributes('data-task-id'))
    expect(rows).toEqual(['3', '2', '1'])
    expect(wrapper.find('.task-group').exists()).toBe(false)
    expect(wrapper.find('.task-group__header').exists()).toBe(false)
    expect(wrapper.find('.task-group__title').exists()).toBe(false)
  })

  it('renders the empty label when no tasks are present', () => {
    const wrapper = mountTasks([])

    expect(wrapper.find('.task-popover__empty').text()).toBe('No active background tasks')
    expect(wrapper.find('.task-popover__body').exists()).toBe(false)
    expect(wrapper.findAll('.task-item')).toHaveLength(0)
  })

  it('gives the whole error as the title of a failed task', () => {
    const error = 'embed batch rpc: internal: opening onnx embedding engine: no runtime'
    const wrapper = mountTasks([{ id: '1', label: 'Indexing', state: 'failed', error }])

    const detail = wrapper.find('.task-item__detail')
    expect(detail.attributes('title')).toBe(error)
    expect(detail.classes()).toContain('task-item__detail--wrapped')
    expect(detail.text()).toBe(error)
  })
})
