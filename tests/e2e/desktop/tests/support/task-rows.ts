/** A list of tasks the test reports to the window as fast as it likes. */
import type { Page } from 'playwright/test'
import type { JsonValue } from './json.js'

/** The names of the wire fields of a task that count how far it has got. */
const WIRE_COUNT = 'done'
const WIRE_TOTAL = 'total'

/**
 * Replaces the window's stream of tasks with one task that is reported again every `every`
 * milliseconds: the source it names changes each time, it names none one time in four, and it counts
 * nothing one time in nine, as the vector pass of the core reports.
 */
export const feedTask = async (page: Page, id: string, every: number): Promise<void> => {
  await page.addInitScript(
    ({ taskId, interval, count, total }) => {
      const encoder = new TextEncoder()
      const frame = (message: JsonValue): Uint8Array => {
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
      const original = window.fetch.bind(window)
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
