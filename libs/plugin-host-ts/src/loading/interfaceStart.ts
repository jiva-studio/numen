/**
 * The start of the plugins that contribute to the interface of a window, before every other plugin.
 * A window is drawn once they have started, or once INTERFACE_WAIT_MS has passed; a plugin of the
 * interface that has not started by then counts as failed.
 */

/** How long a window waits for the plugins of its interface before it is drawn without the late ones. */
export const INTERFACE_WAIT_MS = 3_000

export interface InterfaceStartDeps<Loaded> {
  /** The enabled plugins, in the order they are activated. */
  readonly ids: readonly string[]
  /** Whether the plugin contributes to the interface; false where the manifest cannot be read. */
  readonly isInterface: (id: string) => Promise<boolean>
  readonly load: (id: string) => Promise<Loaded>
  readonly start: (id: string, loaded: Loaded) => Promise<void>
  /** Records a plugin of the interface that has not started within INTERFACE_WAIT_MS as failed. */
  readonly failLate: (id: string) => void
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
 * Starts the plugins of the interface, all read at once and activated in the order of the list,
 * within one INTERFACE_WAIT_MS that also covers telling them apart. Returns the other plugins, in
 * their order, for the start that follows.
 */
export async function startInterfacePlugins<Loaded>(
  deps: InterfaceStartDeps<Loaded>,
): Promise<readonly string[]> {
  const deadline = createDeadline()
  try {
    const kinds = await Promise.all(
      deps.ids.map((id) =>
        Promise.race([deps.isInterface(id).catch(() => false), deadline.late.then(() => false)]),
      ),
    )
    const first = deps.ids.filter((_, index) => kinds[index])
    const loads = first.map((id) => deps.load(id))
    const started = new Set<string>()
    let isLate = false
    const all = (async () => {
      for (const [index, id] of first.entries()) {
        const loaded = await (loads[index] as Promise<Loaded>)
        if (isLate) return
        await deps.start(id, loaded)
        started.add(id)
      }
    })()
    isLate = (await Promise.race([all, deadline.late])) === 'late'
    if (isLate) first.filter((id) => !started.has(id)).forEach(deps.failLate)
    return deps.ids.filter((_, index) => !kinds[index])
  } finally {
    deadline.cancel()
  }
}
