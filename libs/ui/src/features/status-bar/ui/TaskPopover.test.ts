import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import TaskPopover from './TaskPopover.vue'
import type { StatusBarTask } from '../model/types'

function mountTasks(tasks: StatusBarTask[]) {
  return mount(TaskPopover, { props: { tasks } })
}

describe('TaskPopover', () => {
  it('lists a group of one task as a plain row without a group header', () => {
    const wrapper = mountTasks([
      { id: '1', label: 'Embedding vault', group: 'Indexing', state: 'running' },
    ])

    expect(wrapper.find('.task-group').exists()).toBe(false)
    expect(wrapper.findAll('.task-item')).toHaveLength(1)
    expect(wrapper.text()).not.toContain('Indexing')
  })

  it('draws a group header for a group of two or more tasks', () => {
    const wrapper = mountTasks([
      { id: '1', label: 'Chapter 1', group: 'Indexing', state: 'running' },
      { id: '2', label: 'Chapter 2', group: 'Indexing', state: 'running' },
      { id: '3', label: 'Transcribe', group: 'Audio', state: 'running' },
    ])

    const groups = wrapper.findAll('.task-group')
    expect(groups).toHaveLength(1)
    expect(groups[0]?.text()).toContain('Indexing')
    expect(groups[0]?.findAll('.task-item')).toHaveLength(2)
    expect(wrapper.findAll('.task-item')).toHaveLength(3)
  })

  it('gives the whole error as the title of a failed task', () => {
    const error = 'embed batch rpc: internal: opening onnx embedding engine: no runtime'
    const wrapper = mountTasks([
      { id: '1', label: 'Indexing', group: 'Indexing', state: 'failed', error },
    ])

    const detail = wrapper.find('.task-item__detail')
    expect(detail.attributes('title')).toBe(error)
    expect(detail.classes()).toContain('task-item__detail--wrapped')
    expect(detail.text()).toBe(error)
  })
})
