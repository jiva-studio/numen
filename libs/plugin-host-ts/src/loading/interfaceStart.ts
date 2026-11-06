/**
 * The start of the plugins with an entry in a window slot, before every other plugin.
 * A window is drawn once they have started, or once INTERFACE_WAIT_MS has passed. A plugin of a
 * window slot that is not ready by then goes on starting and adds what it contributes when it is.
 */
import { INTERFACE_WAIT_MS } from './limits'

export { INTERFACE_WAIT_MS } from './limits'

export interface InterfaceStartDeps<Loaded> {
  /** The enabled plugins, in the order they are activated. */
  readonly ids: readonly string[]
  /** Whether the plugin has an entry in a window slot; false where the manifest cannot be read. */
  readonly isInterface: (id: string) => Promise<boolean>
  readonly load: (id: string) => Promise<Loaded>
  /** Starts the plugin; never rejects. */
  readonly start: (id: string, loaded: Loaded) => Promise<void>
}

export interface InterfaceStart {
  /** The plugins with no entry in a window slot, in their order. */
  readonly others: readonly string[]
  /** Settles once every plugin of a window slot has started or failed, late ones included. */
  readonly started: Promise<void>
}

/** A promise that settles with 'late' after INTERFACE_WAIT_MS, and the cancel of its timer. */
function createDeadline(): { readonly late: Promise<'late'>; readonly cancel: () => void } {
  let timer: ReturnType<typeof setTimeout> | undefined
  const late = new Promise<'late'>((resolve) => {
    timer = setTimeout(() => resolve('late'), INTERFACE_WAIT_MS)
  })
  return { late, cancel: () => clearTimeout(timer) }
}

/**
 * Starts the plugins of the window slots, all read at once and activated in the order of the list,
 * each after the one before it until INTERFACE_WAIT_MS has passed, and from then on without waiting
 * for a late one. Settles once they have all started or INTERFACE_WAIT_MS has passed, which also
 * covers telling them apart.
 */
export async function startInterfacePlugins<Loaded>(
  deps: InterfaceStartDeps<Loaded>,
): Promise<InterfaceStart> {
  const deadline = createDeadline()
  try {
    const kinds = await Promise.all(
      deps.ids.map((id) =>
        Promise.race([deps.isInterface(id).catch(() => false), deadline.late.then(() => false)]),
      ),
    )
    const first = deps.ids.filter((_, index) => kinds[index])
    const loads = first.map((id) => deps.load(id))
    let isLate = false
    const runs: Promise<void>[] = []
    for (const [index, id] of first.entries()) {
      const run = (loads[index] as Promise<Loaded>).then((loaded) => deps.start(id, loaded))
      runs.push(run)
      if (!isLate) isLate = (await Promise.race([run, deadline.late])) === 'late'
    }
    const started = Promise.all(runs).then(() => undefined)
    await Promise.race([started, deadline.late])
    return { others: deps.ids.filter((_, index) => !kinds[index]), started }
  } finally {
    deadline.cancel()
  }
}
