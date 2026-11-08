/**
 * The start of the plugins of a window: those of the window slots first, then every other one.
 */
import { startInterfacePlugins } from './interfaceStart'
import { INTERFACE_WAIT_MS } from './limits'
import { createStartLimit, type StartLimit } from './startLimit'

export interface StartLoaderDeps {
  readonly pluginIds: readonly string[]
  readonly listen: (id: string) => void
  readonly isEnabled: (id: string) => boolean
  /** Reads the module of a plugin from the window. */
  readonly load: (id: string) => Promise<unknown>
  /** Whether the plugin contributes to the interface, read from its manifest. */
  readonly isInterface: (id: string) => Promise<boolean>
  /** Called once the plugins of the window slots have started or INTERFACE_WAIT_MS has passed. */
  readonly onInterfaceStarted: () => void
}

/**
 * What reading a plugin gave: its module, or the failure, which the activation reads again and
 * reports. `limit` is the start limit of the plugin, which runs from the moment the read began.
 */
type Loaded = { readonly limit: StartLimit } & (
  { readonly isRead: true; readonly module: unknown } | { readonly isRead: false }
)

type Activate = (id: string, module?: unknown, limit?: StartLimit) => Promise<boolean>

/** A plugin that has no frontend has no module, and nothing to activate. */
const startPlugin = async (activate: Activate, id: string, one: Loaded): Promise<void> => {
  if (one.isRead && one.module === undefined) {
    one.limit.stop()
    return
  }
  await activate(id, one.isRead ? one.module : undefined, one.limit)
}

/** What the read of a plugin gave within INTERFACE_WAIT_MS, or 'late' where it gave nothing yet. */
async function readWithin(load: Promise<Loaded>): Promise<Loaded | 'late'> {
  let timer: ReturnType<typeof setTimeout> | undefined
  const late = new Promise<'late'>((resolve) => {
    timer = setTimeout(() => resolve('late'), INTERFACE_WAIT_MS)
  })
  /* Stryker disable BlockStatement,CallExpression: timer cleanup after Promise.race settles */
  try {
    return await Promise.race([load, late])
  } finally {
    clearTimeout(timer)
  }
  /* Stryker restore BlockStatement,CallExpression */
}

/** The read of a plugin, which settles with what it gave and never rejects. */
const readPlugin = (deps: StartLoaderDeps, id: string): Promise<Loaded> => {
  const limit = createStartLimit()
  return deps.load(id).then(
    (module): Loaded => ({ isRead: true, module, limit }),
    // Stryker disable next-line ObjectLiteral: falsy isRead property is equivalent to isRead: false
    (): Loaded => ({ isRead: false, limit }),
  )
}

/**
 * Reads the plugins of the list at once and activates them in its order, so each plugin registers
 * what it provides in the same order on every start. A plugin that is not read within
 * INTERFACE_WAIT_MS is passed over and activated when it is read; it holds back no other plugin.
 */
async function startInOrder(deps: StartLoaderDeps, activate: Activate, ids: readonly string[]) {
  const loading = new Map<string, Promise<Loaded>>(ids.map((id) => [id, readPlugin(deps, id)]))
  // Stryker disable next-line ArrayDeclaration: Promise.all ignores resolved non-promise items when return value is unused
  const running: Promise<unknown>[] = []
  for (const pluginId of ids) {
    const load = loading.get(pluginId) as Promise<Loaded>
    const outcome = await readWithin(load)
    if (outcome === 'late') {
      running.push(load.then((one) => startPlugin(activate, pluginId, one)))
      continue
    }
    await startPlugin(activate, pluginId, outcome)
  }
  await Promise.all(running)
}

/**
 * Starts the enabled plugins of the window slots first (`startInterfacePlugins`), then every other
 * enabled plugin, while the late plugins of the window slots go on starting. The promise settles when every plugin has been activated or has failed.
 */
export async function startLoader(deps: StartLoaderDeps, activate: Activate): Promise<void> {
  for (const pluginId of deps.pluginIds) deps.listen(pluginId)
  const { others, started } = await startInterfacePlugins({
    ids: deps.pluginIds.filter((id) => deps.isEnabled(id)),
    isInterface: deps.isInterface,
    load: (id) => readPlugin(deps, id),
    start: (id, one) => startPlugin(activate, id, one),
  })
  deps.onInterfaceStarted()
  await Promise.all([started, startInOrder(deps, activate, others)])
}
