import { afterEach, describe, expect, it } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { createFakeApi, settle, SUMMARY, task } from '../../testing/fakeApi'
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
    expect(draw({ ...SUMMARY, isReady: false }).wrapper.find('button').exists()).toBe(false)
    wrapper?.unmount()
    expect(
      draw({ ...SUMMARY, overdueCount: 0, todayCount: 0, noDateCount: 0 }).wrapper.find('button').exists(),
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

    expect(fake.calls.listTasks.map((call) => call.filter.due).sort()).toEqual(['none', 'overdue', 'today'])
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

  it('shows each task with a checkbox that cannot be ticked', async () => {
    const { wrapper: view } = draw(SUMMARY)

    await openPopover(view)

    const boxes = view.findAll('li input[type="checkbox"]')
    expect(boxes).toHaveLength(4)
    for (const box of boxes) expect(box.attributes('disabled')).toBeDefined()
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
