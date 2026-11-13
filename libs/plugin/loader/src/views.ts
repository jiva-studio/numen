/**
 * The views plugins declare for other plugins, and the instances of them mounted into the elements of
 * the plugins that mount them.
 *
 * One plugin at a time draws a capability: a second registration of it throws. A mount waits for a
 * declaration of its capability, which starts the plugin that declares it, and goes back to waiting
 * when the plugin that drew it is disposed. Each view is its own Vue application, given the mounting plugin's context as
 * its `context` prop, behind the stylesheet of the plugin that declared it.
 */
import { createApp, type App, type Component } from 'vue'
import type { Disposable, MountedView } from '@numen/plugin-sdk'
import type { ContributionMatch } from './loading'
import { adoptPluginStyle, getPluginStyle, type StyleRoot } from './styles'

/** A view as the registry keeps it. The registry matches capabilities by name and knows no list of them. */
export interface DeclaredViewRegistration {
  readonly capability: string
  readonly component: Component
}

interface DeclaredView {
  readonly pluginId: string
  readonly vaultPermissions: readonly string[]
  readonly registration: DeclaredViewRegistration
}

interface ViewInstance {
  readonly view: DeclaredView
  readonly app: App
  readonly nodes: readonly HTMLElement[]
  readonly release?: (() => void) | undefined
  readonly exposed: { focus?: () => void }
}

/**
 * What a mounting plugin holds. For a view kind a plugin registers itself (`inherits`), the mount
 * also needs every vault permission the registering plugin holds, because the view reads through
 * the services of that plugin.
 */
export interface MountAccess {
  readonly inherits: boolean
  readonly permissions: readonly string[]
  readonly requirePerm: (permission: string, operation: string) => void
}

interface Mount {
  readonly capability: string
  readonly access: MountAccess
  readonly element: HTMLElement
  readonly context: object
  instance?: ViewInstance | undefined
}

function createRoot(): HTMLElement {
  const root = document.createElement('div')
  root.className = 'plugin-view'
  Object.assign(root.style, {
    display: 'flex',
    flexDirection: 'column',
    flex: '1 1 0%',
    minWidth: '0',
    minHeight: '0',
  })
  return root
}

function createStyle(pluginId: string): HTMLElement[] {
  const href = getPluginStyle(pluginId)
  if (!href) return []
  const link = document.createElement('link')
  link.rel = 'stylesheet'
  link.href = href
  link.dataset['pluginId'] = pluginId
  return [link]
}

function isAllowed(mount: Mount, view: DeclaredView): boolean {
  const { inherits, permissions } = mount.access
  return !inherits || view.vaultPermissions.every((one) => permissions.includes(one))
}

function drawView(mount: Mount, view: DeclaredView): ViewInstance {
  const root = createRoot()
  const release = adoptPluginStyle(view.pluginId, mount.element.getRootNode() as StyleRoot)
  const nodes = release ? [root] : [...createStyle(view.pluginId), root]
  mount.element.append(...nodes)
  const app = createApp(view.registration.component, { context: mount.context })
  const exposed = app.mount(root) as { focus?: () => void }
  return { view, app, nodes, exposed, release }
}

function eraseView(mount: Mount): void {
  if (!mount.instance) return
  mount.instance.app.unmount()
  for (const node of mount.instance.nodes) node.remove()
  mount.instance.release?.()
  mount.instance = undefined
}

/** Starts the plugins that declare a capability under `contributes.viewCapabilities`. */
export interface ViewRegistryDeps {
  readonly startPluginsOf?: (match: ContributionMatch) => Promise<void>
}

/** Starts the plugins that declare the capability; a start reports its own failure, and the mount waits. */
function startDeclaringPlugins(deps: ViewRegistryDeps, capability: string): void {
  deps
    .startPluginsOf?.((manifest) =>
      (manifest.contributes?.viewCapabilities ?? []).includes(capability),
    )
    .catch(() => undefined)
}

export function createViewRegistry(deps: ViewRegistryDeps = {}) {
  const declared: DeclaredView[] = []
  const mounts = new Set<Mount>()

  const draw = (mount: Mount) => {
    const view = declared.find((one) => one.registration.capability === mount.capability)
    if (view && isAllowed(mount, view)) mount.instance = drawView(mount, view)
  }

  const registerView = (
    pluginId: string,
    registration: DeclaredViewRegistration,
    permissions: readonly string[] = [],
  ): Disposable => {
    const registrant = declared.find(
      (one) => one.registration.capability === registration.capability,
    )
    if (registrant) {
      throw new Error(
        `Plugin "${pluginId}" registers a view of "${registration.capability}", which plugin "${registrant.pluginId}" already draws`,
      )
    }
    const vaultPermissions = permissions.filter((one) => one.startsWith('vault:'))
    const view: DeclaredView = { pluginId, vaultPermissions, registration }
    declared.push(view)
    for (const mount of mounts) if (!mount.instance) draw(mount)
    return {
      dispose: () => {
        const index = declared.indexOf(view)
        if (index === -1) return
        declared.splice(index, 1)
        for (const mount of mounts) {
          if (mount.instance?.view !== view) continue
          eraseView(mount)
          draw(mount)
        }
      },
    }
  }

  const mountView = (
    capability: string,
    element: HTMLElement,
    ...[context, access]: [context: object, access: MountAccess]
  ): MountedView => {
    const declaredView = declared.find((one) => one.registration.capability === capability)
    for (const permission of access.inherits ? (declaredView?.vaultPermissions ?? []) : []) {
      access.requirePerm(permission, 'mountView')
    }
    const mount: Mount = { capability, element, context, access }
    mounts.add(mount)
    draw(mount)
    if (!declaredView) startDeclaringPlugins(deps, capability)
    return {
      focus: () => mount.instance?.exposed.focus?.(),
      dispose: () => {
        mounts.delete(mount)
        eraseView(mount)
      },
    }
  }

  return { registerView, mountView }
}

export type ViewRegistry = ReturnType<typeof createViewRegistry>
