/**
 * The start of the plugins of a window: those of the interface first, then every other one.
 */
import { startInterfacePlugins } from './interfaceStart'

/** How long the start waits for one plugin to be read before it goes on to the next. */
export const PLUGIN_WAIT_MS = 3_000

export interface StartLoaderDeps {
  readonly pluginIds: readonly string[]
  readonly listen: (id: string) => void
  readonly isEnabled: (id: string) => boolean
  /** Reads the module of a plugin from the window. */
  readonly load: (id: string) => Promise<unknown>
  /** Whether the plugin contributes to the interface, read from its manifest. */
  readonly isInterface: (id: string) => Promise<boolean>
  /** Records a plugin of the interface that did not start within INTERFACE_WAIT_MS as failed. */
  readonly failLate: (id: string) => void
  /** Called once the plugins of the interface have started or INTERFACE_WAIT_MS has passed. */
  readonly onInterfaceStarted: () => void
}

/** What reading a plugin gave: its module, or the failure, which the activation reads again and reports. */
type Loaded = { readonly isRead: true; readonly module: unknown } | { readonly isRead: false }

type Activate = (id: string, module?: unknown) => Promise<boolean>

/** A plugin that has no frontend has no module, and nothing to activate. */
const startPlugin = async (activate: Activate, id: string, one: Loaded): Promise<void> => {
  if (one.isRead && one.module === undefined) return
  await activate(id, one.isRead ? one.module : undefined)
}

/** What the read of a plugin gave within PLUGIN_WAIT_MS, or 'late' where it gave nothing yet. */
async function readWithin(load: Promise<Loaded>): Promise<Loaded | 'late'> {
  let timer: ReturnType<typeof setTimeout> | undefined
  const late = new Promise<'late'>((resolve) => {
    timer = setTimeout(() => resolve('late'), PLUGIN_WAIT_MS)
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
const readPlugin = (deps: StartLoaderDeps, id: string): Promise<Loaded> =>
  deps.load(id).then(
    (module): Loaded => ({ isRead: true, module }),
    // Stryker disable next-line ObjectLiteral: falsy isRead property is equivalent to isRead: false
    (): Loaded => ({ isRead: false }),
  )

/**
 * Reads the plugins of the list at once and activates them in its order, so each plugin registers
 * what it provides in the same order on every start. A plugin that is not read within
 * PLUGIN_WAIT_MS is passed over and activated when it is read; it holds back no other plugin.
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
 * Starts the enabled plugins of the interface first (`startInterfacePlugins`), then every other
 * enabled plugin. The promise settles when every plugin has been activated or has failed.
 */
export async function startLoader(deps: StartLoaderDeps, activate: Activate): Promise<void> {
  for (const pluginId of deps.pluginIds) deps.listen(pluginId)
  const others = await startInterfacePlugins({
    ids: deps.pluginIds.filter((id) => deps.isEnabled(id)),
    isInterface: deps.isInterface,
    load: (id) => readPlugin(deps, id),
    start: (id, one) => startPlugin(activate, id, one),
    failLate: deps.failLate,
  })
  deps.onInterfaceStarted()
  await startInOrder(deps, activate, others)
}
