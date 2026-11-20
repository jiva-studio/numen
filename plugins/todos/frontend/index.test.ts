import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import type { WorkspaceHost } from '@numen/plugin-sdk'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { createFakeApi, createFakeTimer, settle, SUMMARY, task } from './testing/fakeApi'
import { activate, TODOS_VIEW } from './index'

const TODAY = '2026-11-14'

const setup = () => {
  const fake = createFakeApi({
    tasks: {
      overdue: {
        tasks: [
          task({ path: 'Projects/Home.md', line: 7, text: 'fix the tap', due: '2026-11-10' }),
        ],
        total: 1,
        revision: 1,
      },
    },
  })
  const clock = createFakeTimer()
  const open = vi.fn()
  const workspace = {
    window: { getToday: () => TODAY },
    files: { open },
    messages: { showError: vi.fn() },
  } as unknown as WorkspaceHost
  const context = createMockPluginContext({ pluginId: 'md.numen.todos', workspace })
  activate(context, { api: fake.api, timer: clock.timer })
  return { context, fake, clock, open }
}

let view: VueWrapper | null = null

afterEach(() => {
  view?.unmount()
  view = null
})

describe('the todos frontend', () => {
  it('registers the status view the manifest declares and nothing else', () => {
    const { context } = setup()

    expect(TODOS_VIEW).toBe('status')
    expect(context.getRegisteredSlotViews().map((one) => one.id)).toEqual(['status'])
    expect(context.getRegisteredTabKinds()).toEqual([])
    expect(context.getRegisteredCommands()).toEqual([])
  })

  it('refuses to activate in a window that offers no workspace', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.todos' })

    expect(() =>
      activate(context, { api: createFakeApi().api, timer: createFakeTimer().timer }),
    ).toThrow()
  })

  it('follows the tasks of the backend and reads the counts at the day of the window', async () => {
    const { fake } = setup()
    fake.push({ revision: 1, isReady: true })
    await settle()

    expect(fake.calls.watchStarts).toBe(1)
    expect(fake.calls.getSummary).toEqual([TODAY])
  })

  it('draws the count of overdue tasks and tasks due today in the view', async () => {
    const { context, fake } = setup()
    fake.push({ revision: 1, isReady: true })
    await settle()

    view = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })

    expect(view.find('button').text()).toContain(String(SUMMARY.overdueCount + SUMMARY.todayCount))
  })

  it('opens the note at the line of a task clicked in the popover', async () => {
    const { context, fake, open } = setup()
    fake.push({ revision: 1, isReady: true })
    await settle()
    view = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })
    await view.find('button').trigger('click')
    await settle()

    const text = view.findAll('li button').find((one) => one.text() === 'fix the tap')
    await text?.trigger('click')

    expect(open).toHaveBeenCalledWith('Projects/Home.md', { line: 7 })
  })

  it('sends the changed paths of the vault once, after 400 milliseconds', () => {
    const { context, fake, clock } = setup()

    context.events.emitVaultChange({ paths: ['Daily.md'] })
    context.events.emitVaultChange({ paths: ['Projects/Home.md'] })
    expect(fake.calls.refreshIndex).toEqual([])
    expect(clock.waits.map((one) => one.delayMs)).toEqual([400])
    clock.fire()

    expect(fake.calls.refreshIndex).toEqual([
      { paths: ['Daily.md', 'Projects/Home.md'], renamed: [] },
    ])
  })

  it('asks for a full check at once when the vault reports a change with no paths', () => {
    const { context, fake } = setup()

    context.events.emitVaultChange({})

    expect(fake.calls.refreshIndex).toEqual([{ paths: [], renamed: [] }])
  })

  it('stops following the backend and sends nothing more when the plugin stops', async () => {
    const { context, fake, clock } = setup()
    fake.push({ revision: 1, isReady: true })
    await settle()
    context.events.emitVaultChange({ paths: ['Daily.md'] })

    context.dispose()
    await settle()
    clock.fire()
    fake.push({ revision: 2, isReady: true })
    await settle()

    expect(fake.calls.watchEnds).toBe(1)
    expect(fake.calls.refreshIndex).toEqual([])
    expect(fake.calls.getSummary).toEqual([TODAY])
  })

  it('updates the chip and the open popover once the backend has checked a changed note', async () => {
    const { context, fake, clock } = setup()
    fake.push({ revision: 1, isReady: true })
    await settle()
    view = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })
    await view.find('button').trigger('click')
    await settle()
    expect(view.findAll('li')).toHaveLength(1)

    fake.setSummary({ ...SUMMARY, overdueCount: 0, todayCount: 1 })
    fake.setTasks({
      today: {
        tasks: [task({ path: 'Daily.md', line: 2, text: 'water the plants', due: TODAY })],
        total: 1,
        revision: 2,
      },
    })
    context.events.emitVaultChange({ paths: ['Projects/Home.md'] })
    clock.fire()
    await settle()
    await settle()

    expect(fake.calls.getSummary).toEqual([TODAY, TODAY])
    expect(view.find('button').text().trim()).toBe('1')
    const rows = view.findAll('li')
    expect(rows).toHaveLength(1)
    expect(rows[0]?.text()).toContain('water the plants')
  })
})
