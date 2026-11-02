/**
 * A list of tasks the test reports to the window as fast as it likes, and a recording of one row of
 * it while the pointer rests there, for the tests of what the row does when its task is reported
 * again.
 */
import type { Page } from 'playwright/test'

/** The names of the wire fields of a task that count how far it has got. */
const WIRE_COUNT = 'done'
const WIRE_TOTAL = 'total'

/** What the page records for each frame the pointer rests on the row. */
export interface RowRecord {
  readonly frames: number
  readonly updates: number
  /** The distinct states of the ring and the button, each as opacity and whether the row is hovered. */
  readonly looks: readonly string[]
  /** The distinct heights of the row. */
  readonly heights: readonly number[]
  readonly isSameRow: boolean
  readonly isSameRing: boolean
  readonly isSameButton: boolean
  readonly isSameSlot: boolean
  /** Added or removed elements in the popover, by the tag and class of each. */
  readonly changes: readonly string[]
}

/**
 * Replaces the window's stream of tasks with one task that is reported again every `every`
 * milliseconds: the source it names changes each time, it names none one time in four, and it counts
 * nothing one time in nine, as the vector pass of the core reports.
 */
export const feedTask = async (page: Page, id: string, every: number): Promise<void> => {
  await page.addInitScript(
    ({ taskId, interval, count, total }) => {
      const encoder = new TextEncoder()
      const frame = (message: unknown): Uint8Array => {
        const body = encoder.encode(JSON.stringify(message))
        const out = new Uint8Array(5 + body.length)
        new DataView(out.buffer).setUint32(1, body.length)
        out.set(body, 5)
        return out
      }
      const state = { reported: 0, calls: 0 }
      const names = ['Ganguli', 'Mahabharata', 'Vol. 2, Books 4-7', 'Virata Parva']
      const read = () => {
        state.reported += 37
        state.calls += 1
        const hasSource = state.calls % 4 !== 0
        const hasTotal = state.calls % 9 !== 0
        const name = names[state.calls % names.length]
        return {
          id: taskId,
          title: 'Indexing',
          about: hasSource ? `Books/Mahabharata/${name}/part ${state.calls}.md` : '',
          [count]: hasTotal ? String(state.reported) : '0',
          [total]: hasTotal ? '124000' : '0',
          unit: 'UNIT_THINGS',
          isCancellable: true,
        }
      }
      Object.assign(window, { __feed: state })
      const original = window.fetch
      window.fetch = (input, init) => {
        const url =
          typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
        if (!url.endsWith('/numen.v1.WindowService/WatchTasks')) return original(input, init)
        const body = new ReadableStream<Uint8Array>({
          start(controller) {
            controller.enqueue(frame({ tasks: [] }))
            controller.enqueue(frame({ tasks: [read()] }))
            setInterval(() => controller.enqueue(frame({ tasks: [read()] })), interval)
          },
        })
        return Promise.resolve(
          new Response(body, { headers: { 'Content-Type': 'application/connect+json' } }),
        )
      }
    },
    { taskId: id, interval: every, count: WIRE_COUNT, total: WIRE_TOTAL },
  )
}

/** Records the row of the task, and the popover around it, for that many milliseconds of frames. */
export const recordRow = (page: Page, id: string, span: number): Promise<RowRecord> =>
  page.evaluate(
    async ({ taskId, rest }): Promise<RowRecord> => {
      const find = () => document.querySelector<HTMLElement>(`.task-item[data-task-id="${taskId}"]`)
      const row = find()
      const ring = row?.querySelector('.task-item__indicator')
      const button = row?.querySelector('.task-item__btn')
      const slot = row?.querySelector('.task-item__slot')
      const popover = document.querySelector('.task-popover')
      if (!row || !ring || !button || !slot || !popover) throw new Error('the row is not drawn')
      const feed: { calls: number } = Reflect.get(window, '__feed')
      const startedAt = feed.calls

      const changes: string[] = []
      const name = (node: Node): string =>
        node instanceof Element ? `${node.tagName}.${node.getAttribute('class') ?? ''}` : ''
      const note = (verb: string, nodes: NodeList) => {
        for (const node of nodes) {
          if (node.nodeType === 1) changes.push(`${verb} ${name(node)}`)
        }
      }
      const observer = new MutationObserver((list) => {
        for (const one of list) {
          note('added', one.addedNodes)
          note('removed', one.removedNodes)
        }
      })
      observer.observe(popover, { subtree: true, childList: true })

      const look = (): string => {
        const now = find()
        const opacity = (selector: string) => {
          const part = now?.querySelector(selector)
          return part ? getComputedStyle(part).opacity : 'gone'
        }
        const hover = now ? String(now.matches(':hover')) : 'gone'
        return `ring ${opacity('.task-item__indicator')}, button ${opacity('.task-item__btn')}, hover ${hover}`
      }
      const looks = new Set<string>()
      const heights = new Set<number>()
      let frames = 0
      const end = performance.now() + rest
      while (performance.now() < end) {
        looks.add(look())
        heights.add(row.getBoundingClientRect().height)
        frames += 1
        await new Promise((resolve) => requestAnimationFrame(resolve))
      }
      observer.disconnect()

      const now = find()
      const isSame = (part: Element, selector: string) =>
        now?.querySelector(selector) === part && part.isConnected
      return {
        frames,
        updates: feed.calls - startedAt,
        looks: [...looks],
        heights: [...heights],
        isSameRow: now === row && row.isConnected,
        isSameRing: isSame(ring, '.task-item__indicator'),
        isSameButton: isSame(button, '.task-item__btn'),
        isSameSlot: isSame(slot, '.task-item__slot'),
        changes,
      }
    },
    { taskId: id, rest: span },
  )
