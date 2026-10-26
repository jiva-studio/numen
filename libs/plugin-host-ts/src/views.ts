/**
 * The views plugins declare for other plugins, and the instances of them mounted into the elements of
 * the plugins that mount them.
 *
 * A mount waits for a declaration of its capability, and goes back to waiting when the plugin that
 * drew it is disposed. Each view is its own Vue application, given the mounting plugin's context as
 * its `context` prop, behind the stylesheet of the plugin that declared it.
 */
import { createApp, type App, type Component } from 'vue'
import type { Disposable, MountedView, ViewRegistration } from '@numen/plugin-sdk'
import { adoptPluginStyle, getPluginStyle, type StyleRoot } from './styles'

interface DeclaredView {
  readonly pluginId: string
  readonly registration: ViewRegistration
}

interface ViewInstance {
  readonly view: DeclaredView
  readonly app: App
  readonly nodes: readonly HTMLElement[]
  readonly release?: (() => void) | undefined
  readonly exposed: { focus?: () => void }
}

interface Mount {
  readonly capability: string
  readonly element: HTMLElement
  readonly context: unknown
  instance?: ViewInstance | undefined
}

/** The element a view is mounted in, laid out to fill the element the mounting plugin passed. */
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

/** The stylesheet of the declaring plugin as a link, for a plugin whose sheet the root did not adopt. */
function createStyle(pluginId: string): HTMLElement[] {
  const href = getPluginStyle(pluginId)
  if (!href) return []
  const link = document.createElement('link')
  link.rel = 'stylesheet'
  link.href = href
  link.dataset['pluginId'] = pluginId
  return [link]
}

function drawView(mount: Mount, view: DeclaredView): ViewInstance {
  const root = createRoot()
  const release = adoptPluginStyle(view.pluginId, mount.element.getRootNode() as StyleRoot)
  const nodes = release ? [root] : [...createStyle(view.pluginId), root]
  mount.element.append(...nodes)
  const app = createApp(view.registration.component as Component, { context: mount.context })
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

export function createViewRegistry() {
  const declared: DeclaredView[] = []
  const mounts = new Set<Mount>()

  const draw = (mount: Mount) => {
    const view = declared.find((one) => one.registration.capability === mount.capability)
    if (view) mount.instance = drawView(mount, view)
  }

  const registerView = (pluginId: string, registration: ViewRegistration): Disposable => {
    const view: DeclaredView = { pluginId, registration }
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

  const mountView = (capability: string, element: HTMLElement, context: unknown): MountedView => {
    const mount: Mount = { capability, element, context }
    mounts.add(mount)
    draw(mount)
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
