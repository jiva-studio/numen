/**
 * The start of the plugins of a window: only those of the window slots start with it. Every other
 * plugin starts when something it contributes is used.
 */
import type { PluginModule } from '../module'
import { startInterfacePlugins } from './interfaceStart'
import { createStartLimit, type StartLimit } from './startLimit'

export interface StartLoaderDeps {
  readonly pluginIds: readonly string[]
  readonly listen: (id: string) => void
  readonly isEnabled: (id: string) => boolean
  readonly load: (id: string) => Promise<PluginModule | undefined>
  readonly isInterface: (id: string) => Promise<boolean>
  readonly onInterfaceStarted: () => void
}

/**
 * What reading a plugin gave: its module, or the failure, which the activation reads again and
 * reports. `limit` is the start limit of the plugin, which runs from the moment the read began.
 */
type Loaded = { readonly limit: StartLimit } & (
  { readonly isRead: true; readonly module: PluginModule | undefined } | { readonly isRead: false }
)

type Activate = (id: string, module?: PluginModule, limit?: StartLimit) => Promise<boolean>

const startPlugin = async (activate: Activate, id: string, one: Loaded): Promise<void> => {
  if (one.isRead && one.module === undefined) {
    one.limit.stop()
    return
  }
  await activate(id, one.isRead ? one.module : undefined, one.limit)
}

const readPlugin = (deps: StartLoaderDeps, id: string): Promise<Loaded> => {
  const limit = createStartLimit()
  return deps.load(id).then(
    (module): Loaded => ({ isRead: true, module, limit }),
    // Stryker disable next-line ObjectLiteral: falsy isRead property is equivalent to isRead: false
    (): Loaded => ({ isRead: false, limit }),
  )
}

/**
 * Follows the setting of every plugin and starts the enabled plugins of the window slots
 * (`startInterfacePlugins`). The promise settles when each of them has been activated or has failed.
 */
export async function startLoader(deps: StartLoaderDeps, activate: Activate): Promise<void> {
  for (const pluginId of deps.pluginIds) deps.listen(pluginId)
  const { started } = await startInterfacePlugins({
    ids: deps.pluginIds.filter((id) => deps.isEnabled(id)),
    isInterface: deps.isInterface,
    load: (id) => readPlugin(deps, id),
    start: (id, one) => startPlugin(activate, id, one),
  })
  deps.onInterfaceStarted()
  await started
}
