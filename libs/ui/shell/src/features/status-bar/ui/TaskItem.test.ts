import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import TaskItem from './TaskItem.vue'
import TaskPopover from './TaskPopover.vue'
import type { StatusBarTask } from '../model/types'

describe('TaskItem', () => {
  it('renders action button only when failed and actionLabel is present', async () => {
    const task: StatusBarTask = {
      id: 'task-auth-err',
      label: 'Proofreading a recognition',
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

describe('TaskItem row layout', () => {
  const running: StatusBarTask = {
    id: 'vectors',
    label: 'Indexing',
    about: 'Books/Atlas of Imaginary Rivers.pdf',
    progress: 12,
    state: 'running',
    canCancel: true,
  }

  it('draws the progress and the cancel button in one slot before the pointer reaches the row', () => {
    const wrapper = mount(TaskItem, { props: { task: running } })

    const slot = wrapper.find('.task-item__slot')
    expect(slot.classes()).toContain('task-item__slot--swaps')
    expect(slot.findAll('.task-item__indicator')).toHaveLength(1)
    expect(slot.findAll('button')).toHaveLength(1)
    expect(wrapper.findAll('.task-item__indicator')).toHaveLength(1)
    const cancel = wrapper.find('button[aria-label="Cancel task"]')
    expect(cancel.exists()).toBe(true)
    expect(cancel.attributes('disabled')).toBeUndefined()
  })

  it('keeps the same elements while the pointer enters and leaves the row', async () => {
    const wrapper = mount(TaskItem, { props: { task: running }, attachTo: document.body })
    const before = wrapper.html()
    const indicator = wrapper.find('.task-item__indicator').element
    const cancel = wrapper.find('button[aria-label="Cancel task"]').element

    for (const event of ['mouseenter', 'mouseover', 'mousemove', 'mouseleave'] as const) {
      await wrapper.trigger(event)
      expect(wrapper.html()).toBe(before)
    }
    expect(wrapper.find('.task-item__indicator').element).toBe(indicator)
    expect(wrapper.find('button[aria-label="Cancel task"]').element).toBe(cancel)
    wrapper.unmount()
  })

  it('patches the row in place when progress arrives', async () => {
    const wrapper = mount(TaskPopover, { props: { tasks: [running] } })
    const row = wrapper.find('[data-task-id="vectors"]').element
    const cancel = wrapper.find('button[aria-label="Cancel task"]').element

    await wrapper.setProps({ tasks: [{ ...running, progress: 13 }] })
    await wrapper.setProps({ tasks: [{ ...running, progress: 14, about: 'Books/Second.pdf' }] })

    expect(wrapper.find('[data-task-id="vectors"]').element).toBe(row)
    expect(wrapper.find('button[aria-label="Cancel task"]').element).toBe(cancel)
    expect(wrapper.text()).toContain('Books/Second.pdf')
  })

  it('disables the cancel button while the cancel is in progress, and emits nothing', async () => {
    const wrapper = mount(TaskItem, { props: { task: { ...running, state: 'cancelling' } } })

    const cancel = wrapper.find('button[aria-label="Cancelling"]')
    expect(cancel.exists()).toBe(true)
    expect(cancel.attributes('disabled')).toBeDefined()
    await cancel.trigger('click')
    expect(wrapper.emitted('cancel')).toBeUndefined()
  })

  it('draws no cancel button for a task that cannot be cancelled', () => {
    const wrapper = mount(TaskItem, { props: { task: { ...running, canCancel: false } } })

    expect(wrapper.find('button[aria-label="Cancel task"]').exists()).toBe(false)
    expect(wrapper.find('.task-item__slot').classes()).not.toContain('task-item__slot--swaps')
    expect(wrapper.findAll('.task-item__slot button')).toHaveLength(0)
    expect(wrapper.findAll('.task-item__indicator')).toHaveLength(1)
  })

  it('keeps the pointer on the row through reports of the task, which change no class of the slot', async () => {
    const wrapper = mount(TaskItem, { props: { task: running } })
    const row = wrapper.find('.task-item')

    await row.trigger('pointerenter')
    expect(row.classes()).toContain('task-item--hovered')
    const slotClasses = wrapper.find('.task-item__slot').classes()

    for (let step = 1; step <= 20; step++) {
      await wrapper.setProps({
        task: { ...running, progress: step, about: `Books/File ${step}.pdf` },
      })
      expect(wrapper.find('.task-item').element).toBe(row.element)
      expect(row.classes()).toContain('task-item--hovered')
      expect(wrapper.find('.task-item__slot').classes()).toEqual(slotClasses)
    }

    await row.trigger('pointerleave')
    expect(row.classes()).not.toContain('task-item--hovered')
  })

  it('keeps the last source in the detail line while a running task reports none', async () => {
    const wrapper = mount(TaskItem, { props: { task: { ...running, about: 'Books/First.pdf' } } })
    const detail = wrapper.find('.task-item__detail').element

    await wrapper.setProps({ task: { ...running, about: '' } })
    expect(wrapper.find('.task-item__detail').element).toBe(detail)
    expect(wrapper.find('.task-item__detail').text()).toBe('Books/First.pdf')

    await wrapper.setProps({ task: { ...running, about: 'Books/Second.pdf' } })
    expect(wrapper.find('.task-item__detail').text()).toBe('Books/Second.pdf')
  })

  it('draws no detail line for a running task that has named no source', () => {
    const wrapper = mount(TaskItem, { props: { task: { ...running, about: undefined } } })

    expect(wrapper.find('.task-item__detail').exists()).toBe(false)
  })
})

describe('TaskItem row of a notice', () => {
  const text =
    'md.numen.proofreading is the object of no plugin that is installed, and is ignored, so the settings file keeps what it says until the person removes it'
  const notice: StatusBarTask = { id: 'n1', label: 'Read settings', about: text, state: 'notice' }

  it('draws an information icon and the whole text', () => {
    const wrapper = mount(TaskItem, { props: { task: notice } })

    expect(wrapper.find('.task-item__notice-icon').exists()).toBe(true)
    expect(wrapper.find('.task-item__label').text()).toBe('Read settings')
    expect(wrapper.find('.task-item__detail').text()).toBe(text)
    expect(wrapper.find('.task-item__detail').classes()).toContain('task-item__detail--wrapped')
  })

  it('draws no ring and no spinner', () => {
    const wrapper = mount(TaskItem, { props: { task: notice } })

    expect(wrapper.find('.circular-progress').exists()).toBe(false)
    expect(wrapper.find('.task-item__indicator').exists()).toBe(false)
    expect(wrapper.find('.text-alarm').exists()).toBe(false)
  })

  it('draws a dismiss button with a cross that shows without the pointer on the row', async () => {
    const wrapper = mount(TaskItem, { props: { task: notice } })

    const dismiss = wrapper.find('button[aria-label="Dismiss"]')
    expect(dismiss.exists()).toBe(true)
    expect(dismiss.find('svg').exists()).toBe(true)
    expect(dismiss.classes()).toContain('task-item__btn--visible')
    expect(wrapper.findAll('button')).toHaveLength(1)

    await dismiss.trigger('click')
    expect(wrapper.emitted('dismiss')).toEqual([['n1']])
  })
})
