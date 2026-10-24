import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import TaskItem from './TaskItem.vue'
import type { StatusBarTask } from '../model/types'

describe('TaskItem', () => {
  it('renders action button only when failed and actionLabel is present', async () => {
    const task: StatusBarTask = {
      id: 'task-auth-err',
      label: 'Proofreading a recognition',
      group: 'Text Recognition',
      about: 'Books/Sample.pdf',
      state: 'failed',
      error: 'OAuth session expired',
      actionLabel: 'Configure in settings',
    }

    const wrapper = mount(TaskItem, {
      props: { task },
    })

    const actionButton = wrapper.find('.task-item__action-link')
    expect(actionButton.exists()).toBe(true)
    expect(actionButton.text()).toContain('Configure in settings')

    await actionButton.trigger('click')
    expect(wrapper.emitted('action')).toHaveLength(1)
  })

  it('does not render action button when failed but actionLabel is absent', () => {
    const task: StatusBarTask = {
      id: 'task-file-err',
      label: 'Transcribing a recording',
      group: 'Audio & Transcription',
      about: 'assets/sample.flac',
      state: 'failed',
      error: 'stat assets/sample.flac: no such file or directory',
      actionLabel: undefined,
    }

    const wrapper = mount(TaskItem, {
      props: { task },
    })

    const actionButton = wrapper.find('.task-item__action-link')
    expect(actionButton.exists()).toBe(false)
    expect(wrapper.find('.task-item__failure-row').exists()).toBe(false)
  })

  it('does not render action button when task is running', () => {
    const task: StatusBarTask = {
      id: 'task-running',
      label: 'Transcribing a recording',
      group: 'Audio & Transcription',
      about: 'assets/sample.wav',
      state: 'running',
      actionLabel: 'Configure in settings',
    }

    const wrapper = mount(TaskItem, {
      props: { task },
    })

    expect(wrapper.find('.task-item__action-link').exists()).toBe(false)
  })
})
