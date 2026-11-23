import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { createFakeApi, refuse, settle, SUMMARY, createTask } from '../../testing/fakeApi'
import { taskDoneLabel, WORDS } from '../../words'
import type { Summary } from '../../types'
import type { ShownGroups } from '../model/chip'
import TodosStatus from './TodosStatus.vue'

const TODAY = '2026-11-14'
const ALL_SHOWN: ShownGroups = { overdue: true, today: true, nextSevenDays: true, noDate: true }

const TASKS = {
  overdue: {
    tasks: [
      createTask({ path: 'Projects/Home.md', line: 7, text: 'fix the tap', due: '2026-11-10' }),
      createTask({ path: 'Daily/2026-11-12.md', line: 3, text: 'call the bank', due: '2026-11-12' }),
    ],
    total: 7,
    revision: 4,
  },
  today: {
    tasks: [createTask({ path: 'Daily/2026-11-14.md', line: 2, text: 'write the report', due: TODAY })],
    total: 1,
    revision: 4,
  },
  nextSevenDays: {
    tasks: [
      createTask({ path: 'Daily/2026-11-15.md', line: 5, text: 'plan the trip', due: '2026-11-15' }),
      createTask({ path: 'Daily/2026-11-20.md', line: 1, text: 'send the form', due: '2026-11-20' }),
    ],
    total: 2,
    revision: 4,
  },
  none: {
    tasks: [createTask({ path: 'Ideas.md', line: 12, text: 'learn to juggle' })],
    total: 1,
    revision: 4,
  },
}

let wrapper: VueWrapper | null = null

const draw = (
  summary: Summary | null,
  fake = createFakeApi({ tasks: TASKS }),
  shown: ShownGroups = ALL_SHOWN,
) => {
  wrapper = mount(TodosStatus, {
    props: { summary, api: fake.api, getToday: () => TODAY, shown },
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
  it('shows the count of the tasks of every shown group', () => {
    const { wrapper: view } = draw({ ...SUMMARY, overdueCount: 2, todayCount: 3 })

    expect(view.find('button').text()).toContain('8')
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
      draw({ ...SUMMARY, overdueCount: 0, todayCount: 0, noDateCount: 0, nextSevenDaysCount: 0 })
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
  it('lists Overdue, Today, Next 7 days and No date in that order, each with its total', async () => {
    const { wrapper: view } = draw(SUMMARY)

    await openPopover(view)

    const headings = view.findAll('h3').map((one) => one.text())
    expect(headings).toHaveLength(4)
    expect(headings[0]).toMatch(/^Overdue\b.*7/)
    expect(headings[1]).toMatch(/^Today\b.*1/)
    expect(headings[2]).toMatch(/^Next 7 days\b.*2/)
    expect(headings[3]).toMatch(/^No date\b.*1/)
  })

  it('reads each group with its due filter, the day of the window and a limit of 20', async () => {
    const { wrapper: view, fake } = draw(SUMMARY)

    await openPopover(view)

    expect(fake.calls.listTasks.map((call) => call.filter.due).sort()).toEqual([
      'nextSevenDays',
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
    expect(rows).toHaveLength(6)
    expect(rows[0]?.text()).toContain('fix the tap')
    expect(rows[0]?.text()).toContain('2026-11-10')
    expect(rows[0]?.text()).toContain('Home')
    expect(rows[5]?.text()).toContain('learn to juggle')
    expect(rows[5]?.text()).toContain('Ideas')
  })

  it('shows each task with no recurrence with a checkbox that can be ticked', async () => {
    const { wrapper: view } = draw(SUMMARY)

    await openPopover(view)

    const boxes = view.findAll('li input[type="checkbox"]')
    expect(boxes).toHaveLength(6)
    for (const box of boxes) {
      expect(box.attributes('aria-disabled')).toBe('false')
      expect((box.element as HTMLInputElement).checked).toBe(false)
    }
  })

  it('asks to open the note at the line when the text of a task is clicked', async () => {
    const { wrapper: view } = draw(SUMMARY)
    await openPopover(view)

    const row = view.findAll('li')[1]
    const text = row?.findAll('button').find((one) => one.text() === 'call the bank')
    await text?.trigger('click')

    expect(view.emitted('open')).toEqual([
      [{ path: 'Daily/2026-11-12.md', line: 3, raw: '- [ ] call the bank' }],
    ])
  })
})

describe('the todos popover in the top layer', () => {
  const stubTopLayer = () => {
    const show = vi.fn()
    Object.defineProperty(HTMLElement.prototype, 'showPopover', { value: show, configurable: true })
    return show
  }

  afterEach(() => {
    Reflect.deleteProperty(HTMLElement.prototype, 'showPopover')
    vi.restoreAllMocks()
  })

  it('is a manual popover that opens once, placed under the right edge of the chip', async () => {
    const show = stubTopLayer()
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
      bottom: 30,
      right: 700,
    } as DOMRect)
    const { wrapper: view } = draw(SUMMARY)

    await openPopover(view)

    const popover = view.get('[role="dialog"]')
    expect(popover.attributes('popover')).toBe('manual')
    expect(show).toHaveBeenCalledTimes(1)
    expect(popover.attributes('style')).toMatch(/top:.*30px.*right: \d+px/)
  })

  it('works where the browser has no popover', async () => {
    const { wrapper: view } = draw(SUMMARY)

    await openPopover(view)

    expect(view.find('[role="dialog"]').exists()).toBe(true)
  })

  it('closes on Escape and on a press outside', async () => {
    const { wrapper: view } = draw(SUMMARY)

    await openPopover(view)
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await settle()
    expect(view.find('[role="dialog"]').exists()).toBe(false)

    await openPopover(view)
    document.body.dispatchEvent(new Event('pointerdown', { bubbles: true }))
    await settle()
    expect(view.find('[role="dialog"]').exists()).toBe(false)
  })
})

describe('the groups of the todos popover', () => {
  const only = (group: keyof ShownGroups): ShownGroups => ({
    overdue: false,
    today: false,
    nextSevenDays: false,
    noDate: false,
    [group]: true,
  })
  const counts: Summary = {
    ...SUMMARY,
    overdueCount: 7,
    todayCount: 1,
    nextSevenDaysCount: 2,
    noDateCount: 1,
  }

  it('does not draw, read or count a group that is switched off', async () => {
    const shown = { ...ALL_SHOWN, nextSevenDays: false, noDate: false }
    const { wrapper: view, fake } = draw(counts, undefined, shown)

    await openPopover(view)

    expect(view.findAll('h3').map((one) => one.text().replace(/\s+/g, ' '))).toEqual([
      'Overdue 7',
      'Today 1',
    ])
    expect(fake.calls.listTasks.map((call) => call.filter.due).sort()).toEqual(['overdue', 'today'])
    expect(view.find('button').text()).toContain('8')
    expect(view.text()).not.toContain('plan the trip')
  })

  it('hides the chip when every group with tasks is switched off', () => {
    const { wrapper: view } = draw({ ...SUMMARY, overdueCount: 0, todayCount: 0 }, undefined, {
      ...ALL_SHOWN,
      nextSevenDays: false,
      noDate: false,
    })

    expect(view.find('button').exists()).toBe(false)
  })

  it('shows the rows of the next seven days', async () => {
    const { wrapper: view } = draw(counts, undefined, only('nextSevenDays'))

    await openPopover(view)

    expect(view.findAll('li').map((one) => one.text())).toEqual([
      expect.stringContaining('plan the trip'),
      expect.stringContaining('send the form'),
    ])
    expect(view.find('button').text()).toContain('2')
  })

  it('reads the lists again when a group is switched on while the popover is open', async () => {
    const { wrapper: view } = draw(counts, undefined, only('today'))
    await openPopover(view)

    await view.setProps({ shown: ALL_SHOWN })
    await settle()

    expect(view.findAll('h3')).toHaveLength(4)
  })

  it('keeps the groups apart: each task is in one group, and the chip number is the rows drawn', async () => {
    const fake = createFakeApi({
      tasks: {
        overdue: {
          tasks: [createTask({ path: 'a.md', line: 1, text: 'yesterday', due: '2026-11-13' })],
          total: 1,
          revision: 1,
        },
        today: {
          tasks: [createTask({ path: 'a.md', line: 2, text: 'now', due: TODAY })],
          total: 1,
          revision: 1,
        },
        nextSevenDays: {
          tasks: [
            createTask({ path: 'a.md', line: 3, text: 'tomorrow', due: '2026-11-15' }),
            createTask({ path: 'a.md', line: 4, text: 'in six days', due: '2026-11-20' }),
          ],
          total: 2,
          revision: 1,
        },
        none: { tasks: [createTask({ path: 'a.md', line: 5, text: 'floating' })], total: 1, revision: 1 },
      },
    })
    const { wrapper: view } = draw(
      { ...SUMMARY, overdueCount: 1, todayCount: 1, nextSevenDaysCount: 2, noDateCount: 1 },
      fake,
    )

    await openPopover(view)

    const sections = view.findAll('section').map((section) => ({
      heading: section.find('h3').text().replace(/\s+/g, ' '),
      rows: section.findAll('li').map((row) => row.find('.todos-status__text').text()),
    }))
    expect(sections).toEqual([
      { heading: 'Overdue 1', rows: ['yesterday'] },
      { heading: 'Today 1', rows: ['now'] },
      { heading: 'Next 7 days 2', rows: ['tomorrow', 'in six days'] },
      { heading: 'No date 1', rows: ['floating'] },
    ])
    expect(view.text()).not.toContain('finished')
    expect(Number(view.find('.todos-status__trigger span').text())).toBe(view.findAll('li').length)
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
        isProseLine: false,
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

    expect(fake.calls.listTasks.length).toBe(readsBefore + 4)
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
            createTask({
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

    expect(box.attributes('aria-disabled')).toBe('true')
    box.element.parentElement?.dispatchEvent(new MouseEvent('mouseenter'))
    await settle()
    expect(document.body.textContent).toContain(WORDS.recurringNotTickable)
    expect(fake.calls.setTaskDone).toEqual([])
    expect(boxOf(view, 'fix the tap').attributes('aria-disabled')).toBe('false')
  })

  it('shows why a recurring task cannot be ticked when its checkbox is focused', async () => {
    const fake = createFakeApi({
      tasks: {
        ...TASKS,
        overdue: {
          tasks: [
            createTask({ path: 'Home.md', line: 1, text: 'water the plants', recurrence: 'rec:1w' }),
          ],
          total: 1,
          revision: 4,
        },
      },
    })
    const { wrapper: view } = draw(SUMMARY, fake)
    await openPopover(view)

    boxOf(view, 'water the plants').element.dispatchEvent(
      new FocusEvent('focusin', { bubbles: true }),
    )
    await settle()

    expect(document.body.textContent).toContain(WORDS.recurringNotTickable)
  })

  it('labels each checkbox by its task', async () => {
    const { wrapper: view } = draw(SUMMARY)
    await openPopover(view)

    expect(boxOf(view, 'fix the tap').attributes('aria-label')).toBe(taskDoneLabel('fix the tap'))
    expect(boxOf(view, 'call the bank').attributes('aria-label')).toBe(
      taskDoneLabel('call the bank'),
    )
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
    expect(fake.calls.listTasks.length).toBe(readsBefore + 4)
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

describe('a tick in flight in the todos popover', () => {
  it('disables the checkbox of the task until the backend answers, so a second click sends nothing', async () => {
    const pending: { answer?: () => void } = {}
    const fake = createFakeApi({
      tasks: TASKS,
      setTaskDone: (request) =>
        new Promise((resolve) => {
          pending.answer = () => resolve(refuse(request, 'notFound'))
        }),
    })
    const { wrapper: view } = draw(SUMMARY, fake)
    await openPopover(view)

    await boxOf(view, 'call the bank').trigger('click')
    await settle()
    await boxOf(view, 'call the bank').trigger('click')
    await settle()

    expect(boxOf(view, 'call the bank').attributes('aria-disabled')).toBe('true')
    expect(boxOf(view, 'fix the tap').attributes('aria-disabled')).toBe('false')
    expect(fake.calls.setTaskDone).toHaveLength(1)

    pending.answer?.()
    await settle()

    expect(boxOf(view, 'call the bank').attributes('aria-disabled')).toBe('false')
  })
})
