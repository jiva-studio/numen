import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import TaskPopover from './TaskPopover.vue'
import type { StatusBarTask } from '../model/types'

function mountTasks(tasks: StatusBarTask[]) {
  return mount(TaskPopover, { props: { tasks } })
}

describe('TaskPopover', () => {
  it('lists the tasks of the application first, without a group header', () => {
    const wrapper = mountTasks([
      { id: '1', label: 'Reader task', group: 'Example Reader', state: 'running' },
      { id: '2', label: 'Embedding vault', state: 'running' },
      { id: '3', label: 'Walking the notes', state: 'running' },
    ])

    const rows = wrapper.findAll('.task-item').map((row) => row.attributes('data-task-id'))
    expect(rows).toEqual(['2', '3', '1'])
    const groups = wrapper.findAll('.task-group')
    expect(groups).toHaveLength(1)
    expect(wrapper.text()).not.toContain('Numen')
  })

  it('draws no header over the tasks of the application', () => {
    const wrapper = mountTasks([{ id: '1', label: 'Embedding vault', state: 'running' }])

    expect(wrapper.find('.task-group').exists()).toBe(false)
    expect(wrapper.find('.task-group__header').exists()).toBe(false)
    expect(wrapper.findAll('.task-item')).toHaveLength(1)
  })

  it('draws a header with only the plugin name over the tasks of a plugin, whatever their number', () => {
    const wrapper = mountTasks([
      { id: '1', label: 'Chapter 1', group: 'Example Reader', state: 'running' },
      { id: '2', label: 'Chapter 2', group: 'Example Reader', state: 'running' },
      { id: '3', label: 'Transcribe', group: 'Example Audio', state: 'running' },
    ])

    const groups = wrapper.findAll('.task-group')
    expect(groups.map((one) => one.find('.task-group__header').text())).toEqual([
      'Example Reader',
      'Example Audio',
    ])
    expect(groups[0]?.findAll('.task-item')).toHaveLength(2)
    expect(groups[1]?.findAll('.task-item')).toHaveLength(1)
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
