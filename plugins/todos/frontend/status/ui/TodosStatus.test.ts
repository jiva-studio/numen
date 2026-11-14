import { afterEach, describe, expect, it } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { createFakeApi, refuse, settle, SUMMARY, task } from '../../testing/fakeApi'
import { WORDS } from '../../words'
import type { Summary } from '../../types'
import TodosStatus from './TodosStatus.vue'

const TODAY = '2026-11-14'

const TASKS = {
  overdue: {
    tasks: [
      task({ path: 'Projects/Home.md', line: 7, text: 'fix the tap', due: '2026-11-10' }),
      task({ path: 'Daily/2026-11-12.md', line: 3, text: 'call the bank', due: '2026-11-12' }),
    ],
    total: 7,
    revision: 4,
  },
  today: {
    tasks: [task({ path: 'Daily/2026-11-14.md', line: 2, text: 'write the report', due: TODAY })],
    total: 1,
    revision: 4,
  },
  none: {
    tasks: [task({ path: 'Ideas.md', line: 12, text: 'learn to juggle' })],
    total: 1,
    revision: 4,
  },
}

let wrapper: VueWrapper | null = null

const draw = (summary: Summary | null, fake = createFakeApi({ tasks: TASKS })) => {
  wrapper = mount(TodosStatus, {
    props: { summary, api: fake.api, getToday: () => TODAY },
    attachTo: document.body,
  })
  return { wrapper, fake }
}

const openPopover = async (view: VueWrapper) => {
  await view.find('button').trigger('click')
  await settle()
}

afterEach(() => {
  wrapper?.unmount()
  wrapper = null
})

describe('the todos chip', () => {
  it('shows the count of overdue tasks plus tasks due today', () => {
    const { wrapper: view } = draw({ ...SUMMARY, overdueCount: 2, todayCount: 3 })

    expect(view.find('button').text()).toContain('5')
  })

  it('shows nothing when the counts are not ready or no open task exists', () => {
    expect(draw(null).wrapper.find('button').exists()).toBe(false)
    wrapper?.unmount()
    expect(
      draw({ ...SUMMARY, isReady: false })
        .wrapper.find('button')
        .exists(),
    ).toBe(false)
    wrapper?.unmount()
    expect(
      draw({ ...SUMMARY, overdueCount: 0, todayCount: 0, noDateCount: 0 })
        .wrapper.find('button')
        .exists(),
    ).toBe(false)
  })

  it('reads no list before the popover is opened', async () => {
    const { fake } = draw(SUMMARY)
    await settle()

    expect(fake.calls.listTasks).toEqual([])
  })
})

describe('the todos popover', () => {
  it('lists Overdue, Today and No date in that order, each with its total', async () => {
    const { wrapper: view } = draw(SUMMARY)

    await openPopover(view)

    const headings = view.findAll('h3').map((one) => one.text())
    expect(headings).toHaveLength(3)
    expect(headings[0]).toMatch(/^Overdue\b.*7/)
    expect(headings[1]).toMatch(/^Today\b.*1/)
    expect(headings[2]).toMatch(/^No date\b.*1/)
  })

  it('reads each group with its due filter, the day of the window and a limit of 20', async () => {
    const { wrapper: view, fake } = draw(SUMMARY)

    await openPopover(view)

    expect(fake.calls.listTasks.map((call) => call.filter.due).sort()).toEqual([
      'none',
      'overdue',
      'today',
    ])
    for (const call of fake.calls.listTasks) {
      expect(call.today).toBe(TODAY)
      expect(call.limit).toBe(20)
      expect(call.filter.status).toEqual(['open'])
    }
  })

  it('shows for each task its text, its due date and the title of its note', async () => {
    const { wrapper: view } = draw(SUMMARY)

    await openPopover(view)

    const rows = view.findAll('li')
    expect(rows).toHaveLength(4)
    expect(rows[0]?.text()).toContain('fix the tap')
    expect(rows[0]?.text()).toContain('2026-11-10')
    expect(rows[0]?.text()).toContain('Home')
    expect(rows[3]?.text()).toContain('learn to juggle')
    expect(rows[3]?.text()).toContain('Ideas')
  })

  it('shows each task with no recurrence with a checkbox that can be ticked', async () => {
    const { wrapper: view } = draw(SUMMARY)

    await openPopover(view)

    const boxes = view.findAll('li input[type="checkbox"]')
    expect(boxes).toHaveLength(4)
    for (const box of boxes) {
      expect(box.attributes('disabled')).toBeUndefined()
      expect((box.element as HTMLInputElement).checked).toBe(false)
    }
  })

  it('asks to open the note at the line when the text of a task is clicked', async () => {
    const { wrapper: view } = draw(SUMMARY)
    await openPopover(view)

    const row = view.findAll('li')[1]
    const text = row?.findAll('button').find((one) => one.text() === 'call the bank')
    await text?.trigger('click')

    expect(view.emitted('open')).toEqual([[{ path: 'Daily/2026-11-12.md', line: 3 }]])
  })
})

const boxOf = (view: VueWrapper, text: string) => {
  const row = view.findAll('li').find((one) => one.text().includes(text))
  const box = row?.find('input[type="checkbox"]')
  if (!box) throw new Error(`no checkbox for ${text}`)
  return box
}

describe('ticking a task in the todos popover', () => {
  it('sends the path, the line, the text of the line and the day of the window', async () => {
    const { wrapper: view, fake } = draw(SUMMARY)
    await openPopover(view)

    await boxOf(view, 'call the bank').trigger('click')
    await settle()

    expect(fake.calls.setTaskDone).toEqual([
      {
        path: 'Daily/2026-11-12.md',
        line: 3,
        raw: '- [ ] call the bank',
        isDone: true,
        today: TODAY,
      },
    ])
    expect(view.emitted('open')).toBeUndefined()
  })

  it('reads the lists again after a tick, so the ticked task leaves them', async () => {
    const { wrapper: view, fake } = draw(SUMMARY)
    await openPopover(view)
    const readsBefore = fake.calls.listTasks.length

    await boxOf(view, 'call the bank').trigger('click')
    await settle()

    expect(fake.calls.listTasks.length).toBe(readsBefore + 3)
    expect(view.text()).not.toContain('call the bank')
    expect(view.text()).toContain('fix the tap')
    expect(view.findAll('h3')[0]?.text()).toMatch(/^Overdue\b.*6/)
  })

  it('shows a recurring task with a checkbox that cannot be ticked and says why', async () => {
    const fake = createFakeApi({
      tasks: {
        ...TASKS,
        today: {
          tasks: [
            task({
              path: 'Weekly.md',
              line: 4,
              text: 'water the plants',
              due: TODAY,
              recurrence: 'rec:1w',
              raw: '- [ ] water the plants due:2026-11-14 rec:1w',
            }),
          ],
          total: 1,
          revision: 4,
        },
      },
    })
    const { wrapper: view } = draw(SUMMARY, fake)
    await openPopover(view)

    const box = boxOf(view, 'water the plants')
    await box.trigger('click')
    await settle()

    expect(box.attributes('disabled')).toBeDefined()
    expect(box.attributes('title')).toBe(WORDS.recurringNotTickable)
    expect(fake.calls.setTaskDone).toEqual([])
    expect(boxOf(view, 'fix the tap').attributes('disabled')).toBeUndefined()
  })

  it('asks to show that the note changed when the backend refuses the tick, and reads the lists again', async () => {
    const fake = createFakeApi({
      tasks: TASKS,
      setTaskDone: (request) => refuse(request, 'notFound'),
    })
    const { wrapper: view } = draw(SUMMARY, fake)
    await openPopover(view)
    const readsBefore = fake.calls.listTasks.length

    await boxOf(view, 'call the bank').trigger('click')
    await settle()

    expect(view.emitted('showError')).toEqual([[WORDS.changedNote]])
    expect(fake.calls.listTasks.length).toBe(readsBefore + 3)
    expect((boxOf(view, 'call the bank').element as HTMLInputElement).checked).toBe(false)
  })

  it('asks to show that the tick failed when the backend cannot be reached', async () => {
    const fake = createFakeApi({
      tasks: TASKS,
      setTaskDone: () => ({
        ok: false,
        error: { code: 'unavailable', message: 'the host is gone' },
      }),
    })
    const { wrapper: view } = draw(SUMMARY, fake)
    await openPopover(view)

    await boxOf(view, 'call the bank').trigger('click')
    await settle()

    expect(view.emitted('showError')).toEqual([[WORDS.tickFailed]])
    expect((boxOf(view, 'call the bank').element as HTMLInputElement).checked).toBe(false)
  })
})
