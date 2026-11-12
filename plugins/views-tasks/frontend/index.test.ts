/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { Component } from 'vue'
import { createMockPluginContext, type MockPluginContext } from '@numen/plugin-sdk/testing'
import { activate, TASKS_VIEW } from './index'
import { APPEAR_DELAY_MS, HOLD_MS } from './model/chipDisplay'
import { createTask } from './testing/tasks'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  id: string
  targets: readonly string[]
  contributes: { slots: readonly { slot: string; view?: string; command?: string }[] }
}

const mounted: VueWrapper[] = []

afterEach(() => {
  for (const one of mounted.splice(0)) one.unmount()
})

const activateTasks = (): { context: MockPluginContext; view: Component } => {
  const context = createMockPluginContext({ pluginId: manifest.id })
  activate(context)
  const [registration] = context.getRegisteredSlotViews()
  return { context, view: registration!.component as Component }
}

const mountTasks = (view: Component): VueWrapper => {
  const wrapper = mount(view, { attachTo: document.body })
  mounted.push(wrapper)
  return wrapper
}

const openList = async (wrapper: VueWrapper): Promise<void> => {
  await wrapper.find('.tasks-status__trigger').trigger('click')
}

describe('the manifest', () => {
  it('places the view of the plugin in the status slot of the title bar while a vault is open', () => {
    expect(manifest.id).toBe('md.numen.views.tasks')
    expect(manifest.contributes.slots).toStrictEqual([
      { slot: 'titleBar.status', view: TASKS_VIEW, order: 20, when: 'window.isVaultOpen' },
    ])
  })

  it('runs in the editor, the window that has background tasks', () => {
    expect(manifest.targets).toStrictEqual(['desktop.editor'])
  })

  it('registers its view under the id the manifest places', () => {
    const { context } = activateTasks()

    expect(context.getRegisteredSlotViews().map((one) => one.id)).toStrictEqual([TASKS_VIEW])
  })
})

const settle = async (ms: number): Promise<void> => {
  await flushPromises()
  vi.advanceTimersByTime(ms)
  await flushPromises()
}

describe('the summary', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'performance'] })
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('says Idle with an empty ring while no task runs', () => {
    const { view } = activateTasks()

    const wrapper = mountTasks(view)

    expect(wrapper.find('.tasks-status__summary').text()).toBe('Idle')
    expect(wrapper.find('.tasks-status__trigger').attributes('aria-label')).toBe('No tasks running')
    expect(wrapper.find('.tasks-status__ring-track').exists()).toBe(true)
    expect(wrapper.find('.tasks-status__ring-fill').exists()).toBe(false)
  })

  it('counts the tasks done of all and follows the changes of the list', async () => {
    const { context, view } = activateTasks()
    const wrapper = mountTasks(view)
    const running = (id: string) => createTask({ id, state: 'running' })

    context.work.backgroundTasks.setTasks([running('a'), running('b'), running('c')])
    await settle(APPEAR_DELAY_MS)
    expect(wrapper.find('.tasks-status__summary').text()).toBe('0 of 3')

    context.work.backgroundTasks.setTasks([running('b'), running('c')])
    await settle(HOLD_MS)
    expect(wrapper.find('.tasks-status__summary').text()).toBe('1 of 3')
    expect(wrapper.find('.tasks-status__trigger').attributes('aria-label')).toBe(
      '1 of 3 tasks done',
    )
    expect(wrapper.find('.tasks-status__trigger').attributes('title')).toBe('1 of 3 tasks done')

    context.work.backgroundTasks.setTasks([running('c'), running('d')])
    await settle(HOLD_MS)
    expect(wrapper.find('.tasks-status__summary').text()).toBe('2 of 4')

    context.work.backgroundTasks.setTasks([])
    await settle(HOLD_MS)
    expect(wrapper.find('.tasks-status__summary').text()).toBe('Idle')
  })

  it('fills the ring by the progress of the batch', async () => {
    const { context, view } = activateTasks()
    const wrapper = mountTasks(view)

    context.work.backgroundTasks.setTasks([
      createTask({ id: 'a', state: 'running', progress: { completedCount: 1, total: 2 } }),
    ])
    await settle(APPEAR_DELAY_MS)

    const fill = wrapper.find('.tasks-status__ring-fill')
    const length = Number(fill.attributes('stroke-dasharray'))
    expect(Number(fill.attributes('stroke-dashoffset'))).toBeCloseTo(length / 2)
  })

  it('keeps the text of the chip and tints it when tasks failed and nothing else runs', async () => {
    const { context, view } = activateTasks()
    const wrapper = mountTasks(view)

    context.work.backgroundTasks.setTasks([
      createTask({ id: 'f1', state: 'failed', error: 'disk full' }),
      createTask({ id: 'f2', state: 'failed', error: 'no network' }),
    ])
    await flushPromises()

    expect(wrapper.find('.tasks-status__summary').text()).toBe('Idle')
    expect(wrapper.find('.tasks-status__trigger').attributes('aria-label')).toBe('2 failed')
    expect(wrapper.find('.tasks-status__trigger').classes()).toContain(
      'tasks-status__trigger--failed',
    )
  })

  it('keeps the text of the chip and tints it when some task failed while others run', async () => {
    const { context, view } = activateTasks()
    const wrapper = mountTasks(view)

    context.work.backgroundTasks.setTasks([
      createTask({ id: 'a', state: 'running' }),
      createTask({ id: 'f', state: 'running' }),
      createTask({ id: 'b', state: 'running' }),
    ])
    await settle(APPEAR_DELAY_MS)
    context.work.backgroundTasks.setTasks([
      createTask({ id: 'a', state: 'running' }),
      createTask({ id: 'f', state: 'failed', error: 'x' }),
    ])
    await settle(HOLD_MS)

    expect(wrapper.find('.tasks-status__summary').text()).toBe('1 of 3')
    expect(wrapper.find('.tasks-status__trigger').attributes('aria-label')).toBe(
      '1 of 3 tasks done, 1 failed',
    )
  })

  it('takes the tasks the window lists before the view is drawn', async () => {
    const { context, view } = activateTasks()
    context.work.backgroundTasks.setTasks([createTask({ id: 'a', state: 'running' })])

    const wrapper = mountTasks(view)
    await settle(APPEAR_DELAY_MS)

    expect(wrapper.find('.tasks-status__summary').text()).toBe('0 of 1')
  })
})

describe('the chip', () => {
  it('is flat, still and out of the tab order, and opens on a click', async () => {
    const { context, view } = activateTasks()
    const wrapper = mountTasks(view)
    context.work.backgroundTasks.setTasks([createTask({ id: 'a', state: 'running' })])
    await flushPromises()

    const trigger = wrapper.find('.tasks-status__trigger')
    expect(trigger.attributes('tabindex')).toBe('-1')
    expect(wrapper.find('.tasks-status__badge').exists()).toBe(false)
    expect(wrapper.find('.circular-progress').exists()).toBe(false)
    expect(trigger.classes()).toStrictEqual(['tasks-status__trigger'])
    await trigger.trigger('click')
    expect(wrapper.find('.task-popover').exists()).toBe(true)
  })
})

describe('the list', () => {
  it('opens and closes from the summary, newest task first', async () => {
    const { context, view } = activateTasks()
    context.work.backgroundTasks.setTasks([
      createTask({ id: 'first', label: 'Read Gita.pdf' }),
      createTask({ id: 'second', label: 'Transcribe lecture.mp3' }),
    ])
    const wrapper = mountTasks(view)
    expect(wrapper.find('.task-popover').exists()).toBe(false)

    await openList(wrapper)

    expect(wrapper.find('.task-popover__title').text()).toBe('Background tasks')
    expect(
      wrapper.findAll('.task-item').map((row) => row.attributes('data-task-id')),
    ).toStrictEqual(['second', 'first'])

    await openList(wrapper)
    expect(wrapper.find('.task-popover').exists()).toBe(false)
  })

  it('says that there is nothing to show when the list is empty', async () => {
    const { view } = activateTasks()
    const wrapper = mountTasks(view)

    await openList(wrapper)

    expect(wrapper.find('.task-popover__empty').text()).toBe('No active background tasks')
  })

  it('closes on Escape', async () => {
    const { view } = activateTasks()
    const wrapper = mountTasks(view)
    await openList(wrapper)

    globalThis.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await flushPromises()

    expect(wrapper.find('.task-popover').exists()).toBe(false)
  })

  it('shows what a task is done to and how much of it is done', async () => {
    const { context, view } = activateTasks()
    context.work.backgroundTasks.setTasks([
      createTask({
        id: 'a',
        label: 'Index books',
        about: 'Gita.pdf',
        progress: { completedCount: 3, total: 12 },
      }),
    ])
    const wrapper = mountTasks(view)

    await openList(wrapper)

    const row = wrapper.find('[data-task-id="a"]')
    expect(row.find('.task-item__label').text()).toBe('Index books')
    expect(row.find('.task-item__detail').text()).toBe('Gita.pdf')
    expect(row.find('.circular-progress').attributes('data-progress')).toBe('25')
  })

  it('shows the error of a failed task', async () => {
    const { context, view } = activateTasks()
    context.work.backgroundTasks.setTasks([
      createTask({ id: 'a', state: 'failed', about: 'Gita.pdf', error: 'the file is damaged' }),
    ])
    const wrapper = mountTasks(view)

    await openList(wrapper)

    expect(wrapper.find('[data-task-id="a"] .task-item__detail').text()).toBe('the file is damaged')
  })
})

describe('the actions of a task', () => {
  it('asks the window to cancel a running task that can be cancelled', async () => {
    const { context, view } = activateTasks()
    context.work.backgroundTasks.setTasks([createTask({ id: 'a', allowed: { cancel: true } })])
    const wrapper = mountTasks(view)
    await openList(wrapper)

    await wrapper.find('button[aria-label="Cancel task"]').trigger('click')

    expect(context.work.backgroundTasks.getActions()).toStrictEqual([{ action: 'cancel', id: 'a' }])
  })

  it('offers no cancel for a task that cannot be cancelled', async () => {
    const { context, view } = activateTasks()
    context.work.backgroundTasks.setTasks([createTask({ id: 'a', allowed: { cancel: false } })])
    const wrapper = mountTasks(view)
    await openList(wrapper)

    expect(wrapper.find('button[aria-label="Cancel task"]').exists()).toBe(false)
  })

  it('disables the cancel of a task that is being cancelled', async () => {
    const { context, view } = activateTasks()
    context.work.backgroundTasks.setTasks([
      createTask({ id: 'a', state: 'cancelling', allowed: { cancel: true } }),
    ])
    const wrapper = mountTasks(view)
    await openList(wrapper)

    const button = wrapper.find('button[aria-label="Cancelling"]')
    expect(button.attributes('disabled')).toBeDefined()
  })

  it('keeps the list usable when the window does not cancel the task', async () => {
    const { context, view } = activateTasks()
    context.work.backgroundTasks.cancel = () => Promise.reject(new Error('the core is busy'))
    context.work.backgroundTasks.setTasks([createTask({ id: 'a', allowed: { cancel: true } })])
    const wrapper = mountTasks(view)
    await openList(wrapper)

    await wrapper.find('button[aria-label="Cancel task"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-task-id="a"]').exists()).toBe(true)
  })

  it('retries a failed task that can be retried', async () => {
    const { context, view } = activateTasks()
    context.work.backgroundTasks.setTasks([
      createTask({ id: 'a', state: 'failed', error: 'timeout', allowed: { retry: true } }),
    ])
    const wrapper = mountTasks(view)
    await openList(wrapper)

    await wrapper.find('button[aria-label="Retry task"]').trigger('click')

    expect(context.work.backgroundTasks.getActions()).toStrictEqual([{ action: 'retry', id: 'a' }])
  })

  it('dismisses a failed task that cannot be retried', async () => {
    const { context, view } = activateTasks()
    context.work.backgroundTasks.setTasks([
      createTask({ id: 'a', state: 'failed', error: 'timeout' }),
    ])
    const wrapper = mountTasks(view)
    await openList(wrapper)

    expect(wrapper.find('button[aria-label="Retry task"]').exists()).toBe(false)
    await wrapper.find('button[aria-label="Dismiss"]').trigger('click')

    expect(context.work.backgroundTasks.getActions()).toStrictEqual([
      { action: 'dismiss', id: 'a' },
    ])
  })

  it('dismisses a notice', async () => {
    const { context, view } = activateTasks()
    context.work.backgroundTasks.setTasks([
      createTask({ id: 'n', state: 'notice', label: 'The vault was moved' }),
    ])
    const wrapper = mountTasks(view)
    await openList(wrapper)

    expect(wrapper.find('[data-task-id="n"] .task-item__notice-icon').exists()).toBe(true)
    await wrapper.find('[data-task-id="n"] button[aria-label="Dismiss"]').trigger('click')

    expect(context.work.backgroundTasks.getActions()).toStrictEqual([
      { action: 'dismiss', id: 'n' },
    ])
  })

  it('opens the settings of the plugin a failed task names', async () => {
    const { context, view } = activateTasks()
    context.work.backgroundTasks.setTasks([
      createTask({
        id: 'a',
        state: 'failed',
        error: 'api key missing',
        action: { label: 'Configure', pluginId: 'org.example.agent' },
      }),
    ])
    const wrapper = mountTasks(view)
    await openList(wrapper)

    const link = wrapper.find('.task-item__action-link')
    expect(link.text()).toContain('Configure')
    await link.trigger('click')

    expect(context.work.backgroundTasks.getActions()).toStrictEqual([
      { action: 'openSettings', id: 'org.example.agent' },
    ])
  })
})

describe('a window without background tasks', () => {
  it('draws nothing', () => {
    const context = createMockPluginContext({ pluginId: manifest.id })
    const { backgroundTasks: _backgroundTasks, ...work } = context.work
    activate({ ...context, work })
    const [registration] = context.getRegisteredSlotViews()

    const wrapper = mountTasks(registration!.component as Component)

    expect(wrapper.find('.tasks-status').exists()).toBe(false)
  })
})
