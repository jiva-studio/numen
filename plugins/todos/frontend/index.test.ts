import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import {
  CompletionContext,
  type CompletionResult,
  type CompletionSource,
} from '@codemirror/autocomplete'
import { EditorState, type Extension } from '@codemirror/state'
import type { WorkspaceHost } from '@numen/plugin-sdk'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import {
  createFakeApi,
  createFakeTimer,
  refuse,
  settle,
  SUMMARY,
  createTask,
  type FakeInit,
} from './testing/fakeApi'
import { WORDS } from './words'
import { activate, TODOS_VIEW } from './index'

const TODAY = '2026-11-14'

const setup = (init: Pick<FakeInit, 'setTaskDone'> = {}) => {
  const fake = createFakeApi({
    ...init,
    tasks: {
      overdue: {
        tasks: [
          createTask({
            path: 'Projects/Home.md',
            line: 7,
            proseLine: 4,
            text: 'fix the tap',
            due: '2026-11-10',
          }),
        ],
        total: 1,
        revision: 1,
      },
    },
  })
  const clock = createFakeTimer()
  const open = vi.fn()
  const showError = vi.fn()
  const workspace = {
    window: { getToday: () => TODAY },
    files: { open },
    messages: { showError },
  } as unknown as WorkspaceHost
  const context = createMockPluginContext({ pluginId: 'md.numen.todos' })
  const host = createMockHostServices({ workspace })
  activate(context, host, { api: fake.api, timer: clock.timer })
  return { context, host, fake, clock, open, showError }
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
      activate(context, createMockHostServices(), {
        api: createFakeApi().api,
        timer: createFakeTimer().timer,
      }),
    ).toThrow()
  })

  it('follows the tasks of the backend and reads the counts at the day of the window', async () => {
    const { fake } = setup()
    fake.push({ revision: 1, isReady: true })
    await settle()

    expect(fake.calls.watchStarts).toBe(1)
    expect(fake.calls.getSummary).toEqual([TODAY])
  })

  it('draws the count of the shown groups in the view', async () => {
    const { context, fake } = setup()
    fake.push({ revision: 1, isReady: true })
    await settle()

    view = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })

    expect(view.find('button').text()).toContain(
      String(
        SUMMARY.overdueCount +
          SUMMARY.todayCount +
          SUMMARY.nextSevenDaysCount +
          SUMMARY.noDateCount,
      ),
    )
  })

  it('counts only the groups the settings show, and follows a change at once', async () => {
    const { context, fake } = setup()
    fake.push({ revision: 1, isReady: true })
    await settle()
    view = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })

    context.setSetting('showNoDate', false)
    context.setSetting('showNextWeek', false)
    await settle()

    expect(view.find('button').text()).toContain(String(SUMMARY.overdueCount + SUMMARY.todayCount))
  })

  it('opens the note on the 0-based prose line of a task on file line 7 below 3 lines of frontmatter', async () => {
    const { context, fake, open } = setup()
    fake.push({ revision: 1, isReady: true })
    await settle()
    view = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })
    await view.find('button').trigger('click')
    await settle()

    fake.setTasks({
      any: {
        tasks: [
          createTask({ path: 'Projects/Home.md', line: 7, proseLine: 4, text: 'fix the tap' }),
        ],
        total: 1,
        revision: 2,
      },
    })
    const text = view.findAll('li button').find((one) => one.text() === 'fix the tap')
    await text?.trigger('click')
    await settle()

    expect(open).toHaveBeenCalledWith('Projects/Home.md', { line: 3 })
  })

  const clickTask = async (
    fake: ReturnType<typeof setup>['fake'],
    context: { getRegisteredSlotViews: () => { component: unknown }[] },
  ) => {
    fake.push({ revision: 1, isReady: true })
    await settle()
    view = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })
    await view.find('button').trigger('click')
    await settle()
    await view
      .findAll('li button')
      .find((one) => one.text() === 'fix the tap')
      ?.trigger('click')
    await settle()
  }

  it('opens on the line the note holds the task on now, after reading the note again', async () => {
    const { context, fake, open } = setup()
    const tap = (line: number) =>
      createTask({ path: 'Projects/Home.md', line, text: 'fix the tap' })
    fake.push({ revision: 1, isReady: true })
    await settle()
    view = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })
    await view.find('button').trigger('click')
    await settle()
    fake.setTasks({ any: { tasks: [tap(2), tap(11), tap(40)], total: 3, revision: 2 } })
    await view
      .findAll('li button')
      .find((one) => one.text() === 'fix the tap')
      ?.trigger('click')
    await settle()

    expect(fake.calls.refreshIndex).toContainEqual({ paths: ['Projects/Home.md'], renamed: [] })
    expect(fake.calls.listTasks.at(-1)?.filter.path).toBe('Projects/Home.md')
    expect(open).toHaveBeenCalledWith('Projects/Home.md', { line: 10 })
  })

  it('opens the note without a line where the note cannot be read again', async () => {
    const { context, fake, open } = setup()
    fake.api.refreshIndex = async () => {
      throw new Error('gone')
    }
    await clickTask(fake, context)

    expect(open).toHaveBeenCalledWith('Projects/Home.md', undefined)
  })

  it('says so once where the note cannot be read again, and still opens it', async () => {
    const { context, fake, open, showError } = setup()
    fake.api.refreshIndex = async () => {
      throw new Error('gone')
    }
    await clickTask(fake, context)

    expect(showError).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith(WORDS.taskNotRead)
    expect(open).toHaveBeenCalledWith('Projects/Home.md', undefined)
  })

  it('says so where the note cannot be opened', async () => {
    const { context, fake, open, showError } = setup()
    open.mockImplementation(() => {
      throw new Error('no such note')
    })
    await clickTask(fake, context)

    expect(showError).toHaveBeenCalledWith(WORDS.noteNotOpened)
  })

  it('opens the note without a line where the task has no prose line', async () => {
    const { context, fake, open } = setup()
    fake.push({ revision: 1, isReady: true })
    await settle()
    view = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })
    await view.find('button').trigger('click')
    await settle()
    fake.setTasks({
      any: {
        tasks: [
          createTask({ path: 'Projects/Home.md', line: 7, proseLine: 0, text: 'fix the tap' }),
        ],
        total: 1,
        revision: 2,
      },
    })
    await view
      .findAll('li button')
      .find((one) => one.text() === 'fix the tap')
      ?.trigger('click')
    await settle()

    expect(open).toHaveBeenCalledWith('Projects/Home.md', undefined)
  })

  it('ticks a task in the popover and reads the counts again when the backend reports the change', async () => {
    const { context, fake } = setup()
    fake.push({ revision: 1, isReady: true })
    await settle()
    view = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })
    await view.find('button').trigger('click')
    await settle()

    await view.find('li input[type="checkbox"]').trigger('click')
    await settle()
    fake.push({ revision: 2, isReady: true })
    await settle()

    expect(fake.calls.setTaskDone.map((one) => [one.path, one.line, one.isDone])).toEqual([
      ['Projects/Home.md', 7, true],
    ])
    expect(fake.calls.getSummary).toEqual([TODAY, TODAY])
  })

  it('shows the message of a refused tick in the window', async () => {
    const { context, fake, showError } = setup({
      setTaskDone: (request) => refuse(request, 'stale'),
    })
    fake.push({ revision: 1, isReady: true })
    await settle()
    view = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })
    await view.find('button').trigger('click')
    await settle()

    await view.find('li input[type="checkbox"]').trigger('click')
    await settle()

    expect(showError).toHaveBeenCalledWith(WORDS.changedNote)
  })

  it('sends the changed paths of the vault once, after 400 milliseconds', () => {
    const { host, fake, clock } = setup()

    host.events.emitVaultChange({ paths: ['Daily.md'] })
    host.events.emitVaultChange({ paths: ['Projects/Home.md'] })
    expect(fake.calls.refreshIndex).toEqual([])
    expect(clock.waits.map((one) => one.delayMs)).toEqual([400])
    clock.fire()

    expect(fake.calls.refreshIndex).toEqual([
      { paths: ['Daily.md', 'Projects/Home.md'], renamed: [] },
    ])
  })

  it('asks for a full check at once when the vault reports a change with no paths', () => {
    const { host, fake } = setup()
    host.events.emitVaultChange({})

    expect(fake.calls.refreshIndex).toEqual([{ paths: [], renamed: [] }])
  })

  it('stops following the backend and sends nothing more when the plugin stops', async () => {
    const { context, host, fake, clock } = setup()
    fake.push({ revision: 1, isReady: true })
    await settle()
    host.events.emitVaultChange({ paths: ['Daily.md'] })

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
    const { context, host, fake, clock } = setup()
    fake.push({ revision: 1, isReady: true })
    await settle()
    view = mount(context.getRegisteredSlotViews()[0]?.component as never, {
      attachTo: document.body,
    })
    await view.find('button').trigger('click')
    await settle()
    expect(view.findAll('li')).toHaveLength(1)

    fake.setSummary({
      ...SUMMARY,
      overdueCount: 0,
      todayCount: 1,
      nextSevenDaysCount: 0,
      noDateCount: 0,
    })
    fake.setTasks({
      today: {
        tasks: [createTask({ path: 'Daily.md', line: 2, text: 'water the plants', due: TODAY })],
        total: 1,
        revision: 2,
      },
    })
    host.events.emitVaultChange({ paths: ['Projects/Home.md'] })
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

describe('the editor extension of the todos plugin', () => {
  it('registers the extension the manifest declares', () => {
    const { context } = setup()

    expect(context.getRegisteredEditorExtensions().map((one) => one.id)).toEqual(['tasks'])
  })
})

describe('the task completion of the todos plugin', () => {
  const projects = [{ name: 'home', count: 2 }]

  const complete = async (extension: Extension, doc: string) => {
    const state = EditorState.create({ doc, extensions: [extension] })
    const source = state
      .languageDataAt<CompletionSource>('autocomplete', doc.length)
      .find((one) => one !== undefined)
    return (await source?.(
      new CompletionContext(state, doc.length, true),
    )) as CompletionResult | null
  }

  it('reads the fields once at activation, before the stream sends a message', async () => {
    const fake = createFakeApi({ fields: { projects, contexts: [], tags: [] } })
    const context = createMockPluginContext({ pluginId: 'md.numen.todos' })
    const workspace = {
      window: { getToday: () => TODAY },
      files: { open: vi.fn() },
      messages: { showError: vi.fn() },
    } as unknown as WorkspaceHost
    activate(context, createMockHostServices({ workspace }), {
      api: fake.api,
      timer: createFakeTimer().timer,
    })
    await settle()

    const extension = context.getRegisteredEditorExtensions()[0]?.extension as Extension
    const result = await complete(extension, '- [ ] pay +')
    expect(result?.options.map((one) => one.label)).toEqual(['home'])
  })

  it('reads the fields again when the backend reports the index ready', async () => {
    const fake = createFakeApi({ summary: { ...SUMMARY, isReady: false } })
    const context = createMockPluginContext({ pluginId: 'md.numen.todos' })
    const workspace = {
      window: { getToday: () => TODAY },
      files: { open: vi.fn() },
      messages: { showError: vi.fn() },
    } as unknown as WorkspaceHost
    activate(context, createMockHostServices({ workspace }), {
      api: fake.api,
      timer: createFakeTimer().timer,
    })
    await settle()
    fake.setFields({ projects, contexts: [], tags: [] })
    fake.setSummary({ ...SUMMARY, isReady: true })
    fake.push({ revision: 1, isReady: true })
    await settle()

    const extension = context.getRegisteredEditorExtensions()[0]?.extension as Extension
    expect((await complete(extension, '- [ ] pay +'))?.options).toHaveLength(1)
  })

  it('offers the tags after a hash', async () => {
    const fake = createFakeApi({
      fields: { projects: [], contexts: [], tags: [{ name: 'port', count: 3 }] },
    })
    const context = createMockPluginContext({ pluginId: 'md.numen.todos' })
    const workspace = {
      window: { getToday: () => TODAY },
      files: { open: vi.fn() },
      messages: { showError: vi.fn() },
    } as unknown as WorkspaceHost
    activate(context, createMockHostServices({ workspace }), {
      api: fake.api,
      timer: createFakeTimer().timer,
    })
    await settle()

    const extension = context.getRegisteredEditorExtensions()[0]?.extension as Extension
    expect((await complete(extension, '- [ ] pay #'))?.options.map((one) => one.label)).toEqual([
      'port',
    ])
  })

  it('registers the extension again when the hideFields setting changes', () => {
    const { context } = setup()
    const before = context.getRegisteredEditorExtensions()[0]?.extension

    context.setSetting('hideFields', true)

    const after = context.getRegisteredEditorExtensions()
    expect(after).toHaveLength(1)
    expect(after[0]?.extension).not.toBe(before)
  })

  it('refuses to activate in a host without editor extensions', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.todos' })
    const register = context.register as { editorExtension?: unknown }
    register.editorExtension = undefined
    const workspace = { window: { getToday: () => TODAY } } as unknown as WorkspaceHost

    expect(() =>
      activate(context, createMockHostServices({ workspace }), {
        api: createFakeApi().api,
        timer: createFakeTimer().timer,
      }),
    ).toThrow()
  })
})
