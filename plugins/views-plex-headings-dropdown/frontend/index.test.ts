/**
 * The view of the plugin as the plex draws it over a node: the headings of one note, read when the pointer rests
 * on the node, listed under it, scrolled with the wheel and chosen to open the note at their line.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { DOMWrapper, mount, type VueWrapper } from '@vue/test-utils'
import { nextTick, type Component } from 'vue'
import type { JsonValue, NoteHeading, VaultNotes, WorkspaceHost } from '@numen/plugin-sdk'
import {
  createMockHostServices,
  createMockNoticeHost,
  createMockPluginContext,
} from '@numen/plugin-sdk/testing'
import { activate } from './index'
import { HOVER_WAIT, LEAVE_GRACE } from './model/useHeadings'

const reported = (text: string): Error => Object.assign(new Error(text), { name: 'ReportedError' })

const HEAT = 'physics/Heat.md'
const WORK = 'physics/Work.md'
const WINDOW = { width: 1200, height: 800 }
const NODE = { width: 144, height: 36 }

const sections = (count: number): NoteHeading[] =>
  Array.from({ length: count }, (_, index) => ({
    text: `Section ${index + 1}`,
    level: 1,
    line: 10 * (index + 1),
  }))

const wrappers: VueWrapper[] = []

const setup = (
  headings: Readonly<Record<string, readonly NoteHeading[]>> = { [HEAT]: sections(3) },
  initialSettings: Record<string, JsonValue> = {},
) => {
  Object.assign(window, { innerWidth: WINDOW.width, innerHeight: WINDOW.height })
  const requested: (readonly string[])[] = []
  const answers = new Map<string, Promise<NoteHeading[]> | NoteHeading[]>()
  const notes = {
    getHeadings: async (paths: readonly string[]) => {
      requested.push(paths)
      return new Map(
        await Promise.all(
          paths.map(
            async (path) => [path, await (answers.get(path) ?? headings[path] ?? [])] as const,
          ),
        ),
      )
    },
  } as unknown as VaultNotes
  const opened: [string, number | undefined][] = []
  const workspace = {
    files: {
      open: (path: string, options?: { line?: number }) => opened.push([path, options?.line]),
    },
  } as unknown as WorkspaceHost
  const context = createMockPluginContext({
    pluginId: 'md.numen.views.plex.headings.dropdown',
    initialSettings,
  })
  const notices = createMockNoticeHost()
  const host = createMockHostServices({ notes, workspace, notices })
  activate(context, host)
  const view = context.getRegisteredSlotViews().find((one) => one.id === 'headings')!
    .component as Component
  const show = (path: string, y = 0) => {
    const wrapper = mount(view, {
      props: { path, x: 0, y, ...NODE, scale: 1 },
      attachTo: document.body,
    })
    const left = WINDOW.width / 2 - NODE.width / 2
    const top = WINDOW.height / 2 + y - NODE.height / 2
    vi.spyOn(wrapper.get('.headings').element, 'getBoundingClientRect').mockReturnValue(
      new DOMRect(left, top, NODE.width, NODE.height),
    )
    wrappers.push(wrapper)
    return wrapper
  }
  return { context, host, notices, show, requested, answers, opened }
}

const settle = async (wait = 0) => {
  await vi.advanceTimersByTimeAsync(wait)
  await nextTick()
}

const leave = async (wrapper: VueWrapper) => {
  await wrapper.get('.headings').trigger('pointerleave')
  await settle(LEAVE_GRACE)
}

const rest = async (wrapper: VueWrapper, wait = HOVER_WAIT) => {
  await wrapper.get('.headings').trigger('pointerenter')
  await settle(wait)
}

const getRows = () =>
  Array.from(document.querySelectorAll('.headings__row')).map((one) => new DOMWrapper(one))
const getList = () => document.querySelector<HTMLElement>('.headings__list')
const getTexts = (_wrapper?: VueWrapper) => getRows().map((one) => one.text())

afterEach(() => {
  vi.useRealTimers()
  while (wrappers.length > 0) wrappers.pop()?.unmount()
  document.body.replaceChildren()
})

describe('the plugin', () => {
  it('registers the view its manifest places across the nodes of the plex', () => {
    const { context } = setup()

    expect(context.getRegisteredSlotViews().map((one) => one.id)).toStrictEqual(['headings'])
  })

  it('refuses to activate without the notes of the vault or the window', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.views.plex.headings.dropdown' })

    expect(() => activate(context, createMockHostServices())).toThrow(/need the notes/)
  })
})

describe('a node the pointer rests on', () => {
  it('lists the headings of its note in the order they stand in it, once the pointer has rested', async () => {
    vi.useFakeTimers()
    const { show } = setup()
    const wrapper = show(HEAT)

    await rest(wrapper)

    expect(getTexts(wrapper)).toStrictEqual(['Section 1', 'Section 2', 'Section 3'])
  })

  it('lists nothing before the pointer has rested for the wait', async () => {
    vi.useFakeTimers()
    const { show } = setup()
    const wrapper = show(HEAT)

    await rest(wrapper, HOVER_WAIT - 1)

    expect(getTexts(wrapper)).toStrictEqual([])
  })

  it('reads the headings of that one note and no other', async () => {
    vi.useFakeTimers()
    const { show, requested } = setup({ [HEAT]: sections(3), [WORK]: sections(2) })
    show(WORK)
    const wrapper = show(HEAT)

    expect(requested).toStrictEqual([])
    await rest(wrapper)

    expect(requested).toStrictEqual([[HEAT]])
  })

  it('draws nothing for a note with no heading', async () => {
    vi.useFakeTimers()
    const { show } = setup({})
    const wrapper = show(HEAT)

    await rest(wrapper)

    expect(getList() !== null).toBe(false)
  })

  it('draws the list outside the view, over the window', async () => {
    vi.useFakeTimers()
    const { show } = setup()
    const wrapper = show(HEAT, 50)
    await rest(wrapper)

    expect(wrapper.get('.headings').element.contains(getList())).toBe(false)
    expect(document.body.contains(getList())).toBe(true)
  })

  it('keeps the list while the pointer crosses from the node to it', async () => {
    vi.useFakeTimers()
    const { show } = setup()
    const wrapper = show(HEAT)
    await rest(wrapper)

    await wrapper.get('.headings').trigger('pointerleave')
    await settle(LEAVE_GRACE - 1)
    getList()!.dispatchEvent(new Event('pointerenter'))
    await settle(LEAVE_GRACE)

    expect(getTexts()).toHaveLength(3)
  })

  it('puts the list away once the pointer has left the list too', async () => {
    vi.useFakeTimers()
    const { show } = setup()
    const wrapper = show(HEAT)
    await rest(wrapper)
    getList()!.dispatchEvent(new Event('pointerenter'))

    getList()!.dispatchEvent(new Event('pointerleave'))
    await settle(LEAVE_GRACE)

    expect(getList()).toBeNull()
  })

  it('sets a heading in by its level', async () => {
    vi.useFakeTimers()
    const { show } = setup({
      [HEAT]: [
        { text: 'Laws', level: 1, line: 1 },
        { text: 'First law', level: 2, line: 5 },
      ],
    })
    const wrapper = show(HEAT)

    await rest(wrapper)
    const [outer, inner] = getRows().map((one) => one.attributes('style'))

    expect(outer).toContain('--row-indent: 0px')
    expect(inner).toContain('--row-indent: 12px')
  })

  it('stops at the number of headings the setting allows, and marks that there is more', async () => {
    vi.useFakeTimers()
    const { show } = setup({ [HEAT]: sections(9) }, { headingLimit: 4 })
    const wrapper = show(HEAT)

    await rest(wrapper)

    expect(getTexts(wrapper)).toHaveLength(4)
    expect(document.querySelectorAll('.headings__more')).toHaveLength(1)
  })

  it('stops at the number of headings the room under the node holds', async () => {
    vi.useFakeTimers()
    const { show } = setup({ [HEAT]: sections(9) })
    const wrapper = show(HEAT, 330)

    await rest(wrapper)

    expect(getTexts(wrapper)).toHaveLength(1)
  })

  it('draws no list where there is no room under the node for one row', async () => {
    vi.useFakeTimers()
    const { show } = setup({ [HEAT]: sections(9) })
    const wrapper = show(HEAT, 390)

    await rest(wrapper)

    expect(getList() !== null).toBe(false)
  })
})

describe('a pointer that leaves', () => {
  it('puts the list away', async () => {
    vi.useFakeTimers()
    const { show } = setup()
    const wrapper = show(HEAT)
    await rest(wrapper)

    await leave(wrapper)

    expect(getTexts(wrapper)).toStrictEqual([])
  })

  it('lists nothing for an answer that arrives after the pointer left', async () => {
    vi.useFakeTimers()
    const { show, answers } = setup()
    let answer: (headings: NoteHeading[]) => void = () => undefined
    answers.set(HEAT, new Promise<NoteHeading[]>((resolve) => (answer = resolve)))
    const wrapper = show(HEAT)
    await rest(wrapper)

    await leave(wrapper)
    answer(sections(3))
    await settle()

    expect(getTexts(wrapper)).toStrictEqual([])
  })

  it('lists nothing for a stale answer that arrives after the list was read again', async () => {
    vi.useFakeTimers()
    const { show, answers } = setup()
    let late: (headings: NoteHeading[]) => void = () => undefined
    answers.set(HEAT, new Promise<NoteHeading[]>((resolve) => (late = resolve)))
    const wrapper = show(HEAT)
    await rest(wrapper)
    await leave(wrapper)
    answers.set(HEAT, [{ text: 'Fresh', level: 1, line: 2 }])

    await rest(wrapper)
    late(sections(3))
    await settle()

    expect(getTexts(wrapper)).toStrictEqual(['Fresh'])
  })

  it('reuses the headings when the pointer returns, and reads again after the vault changes', async () => {
    vi.useFakeTimers()
    const { show, requested, host } = setup()
    const wrapper = show(HEAT)

    await rest(wrapper)
    await leave(wrapper)
    await rest(wrapper)

    expect(requested).toStrictEqual([[HEAT]])

    await leave(wrapper)
    host.events.emitVaultChange({ paths: [HEAT] })
    await rest(wrapper)

    expect(requested).toStrictEqual([[HEAT], [HEAT]])
  })
})

describe('the limit of the headings', () => {
  it('follows the limit as the person changes it', async () => {
    vi.useFakeTimers()
    const { show, context } = setup({ [HEAT]: sections(9) })
    const wrapper = show(HEAT)
    await rest(wrapper)

    context.setSetting('headingLimit', 2)
    await settle()

    expect(getTexts(wrapper)).toHaveLength(2)
  })
})

describe('a heading chosen', () => {
  it('opens the note at the line of the heading, in the current pane', async () => {
    vi.useFakeTimers()
    const { show, opened } = setup()
    const wrapper = show(HEAT)
    await rest(wrapper)

    await getRows()[1]!.trigger('click')

    expect(opened).toStrictEqual([[HEAT, 20]])
  })

  it('is not a click on the node', async () => {
    vi.useFakeTimers()
    const { show } = setup()
    const wrapper = show(HEAT)
    await rest(wrapper)
    const reached = vi.fn()
    document.body.addEventListener('click', reached)

    await getRows()[0]!.trigger('click')

    expect(reached).not.toHaveBeenCalled()
    document.body.removeEventListener('click', reached)
  })

  it('is not a double click on the node either', async () => {
    vi.useFakeTimers()
    const { show } = setup()
    const wrapper = show(HEAT)
    await rest(wrapper)
    const reached = vi.fn()
    document.body.addEventListener('dblclick', reached)

    await getRows()[0]!.trigger('dblclick')

    expect(reached).not.toHaveBeenCalled()
    document.body.removeEventListener('dblclick', reached)
  })

  it('is not the only click the node receives: a click on the rest of the view reaches it', async () => {
    vi.useFakeTimers()
    const { show } = setup()
    const wrapper = show(HEAT)
    const reached = vi.fn()
    document.body.addEventListener('click', reached)

    await wrapper.get('.headings').trigger('click')

    expect(reached).toHaveBeenCalledTimes(1)
    document.body.removeEventListener('click', reached)
  })
})

describe('the wheel on the list', () => {
  const wheel = async (_wrapper: VueWrapper, deltaY: number) => {
    getList()!.dispatchEvent(
      new WheelEvent('wheel', { deltaY, deltaMode: 1, bubbles: true, cancelable: true }),
    )
    await nextTick()
  }

  const setupMany = async () => {
    vi.useFakeTimers()
    const { show } = setup({ [HEAT]: sections(9) })
    const wrapper = show(HEAT)
    await rest(wrapper)
    return wrapper
  }

  it('scrolls the window down a heading at a time, and back up again', async () => {
    const wrapper = await setupMany()

    await wheel(wrapper, 1)
    expect(getTexts(wrapper)[0]).toBe('Section 2')
    expect(document.querySelectorAll('.headings__more')).toHaveLength(2)

    await wheel(wrapper, -1)
    expect(getTexts(wrapper)[0]).toBe('Section 1')
  })

  it('scrolls no further than either end of the headings', async () => {
    const wrapper = await setupMany()

    await wheel(wrapper, -1)
    expect(getTexts(wrapper)[0]).toBe('Section 1')

    for (let step = 0; step < 8; step++) await wheel(wrapper, 1)
    expect(getTexts(wrapper).at(-1)).toBe('Section 9')
    expect(document.querySelectorAll('.headings__more')).toHaveLength(1)
  })

  it('keeps the wheel from the plex while it scrolls, and lets it through at either end', async () => {
    await setupMany()
    const down = new WheelEvent('wheel', {
      deltaY: 1,
      deltaMode: 1,
      bubbles: true,
      cancelable: true,
    })
    const up = new WheelEvent('wheel', {
      deltaY: -1,
      deltaMode: 1,
      bubbles: true,
      cancelable: true,
    })

    getList()!.dispatchEvent(up)
    getList()!.dispatchEvent(down)

    expect(up.defaultPrevented).toBe(false)
    expect(down.defaultPrevented).toBe(true)
  })

  it('scrolls nothing where every heading is shown at once', async () => {
    vi.useFakeTimers()
    const { show } = setup()
    const wrapper = show(HEAT)
    await rest(wrapper)

    await wheel(wrapper, 1)

    expect(getTexts(wrapper)[0]).toBe('Section 1')
    expect(document.querySelectorAll('.headings__more')).toHaveLength(0)
  })

  it('opens at the top each time the list is shown', async () => {
    const wrapper = await setupMany()
    await wheel(wrapper, 1)

    await leave(wrapper)
    await rest(wrapper)

    expect(getTexts(wrapper)[0]).toBe('Section 1')
  })
})

describe('a failure to read the headings', () => {
  it('is shown to the person as an error, and no list is shown', async () => {
    vi.useFakeTimers()
    const { show, notices, answers } = setup()
    answers.set(HEAT, Promise.reject(reported('the vault is closed')))
    const wrapper = show(HEAT)

    await rest(wrapper)

    expect(notices.getShown()).toStrictEqual([{ text: 'the vault is closed', severity: 'error' }])
    expect(getTexts(wrapper)).toStrictEqual([])
  })
})
