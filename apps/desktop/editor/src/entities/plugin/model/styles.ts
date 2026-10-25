/**
 * Scoped stylesheet registry for plugin views and Shadow DOM style containment.
 */
import { defineAsyncComponent, h, Teleport, type Component, type Ref, type VNodeChild } from 'vue'
import type { PluginContext } from '@numen/plugin-sdk'

type AsyncComponentRegistration = {
  type: 'async'
  loader: () => Promise<Component>
}

function isAsyncComponent(
  comp: unknown,
): comp is (() => Promise<Component>) | AsyncComponentRegistration {
  if (typeof comp === 'object' && comp !== null && (comp as { type?: unknown }).type === 'async') {
    return typeof (comp as { loader?: unknown }).loader === 'function'
  }
  if (typeof comp !== 'function') return false
  return !(
    'setup' in comp ||
    'render' in comp ||
    'props' in comp ||
    '__name' in comp ||
    (comp as { length?: number }).length! > 0
  )
}

export function resolvePluginComponent(rawComponent: unknown): Component | null {
  if (!rawComponent) return null
  const comp =
    typeof rawComponent === 'object' && 'default' in rawComponent
      ? (rawComponent as { default: unknown }).default
      : rawComponent
  if (!comp) return null
  if (isAsyncComponent(comp)) {
    const loader = typeof comp === 'function' ? comp : comp.loader
    return defineAsyncComponent(loader)
  }
  return comp as Component
}

export function extractTitleValue(title: unknown): string {
  if (typeof title === 'string') return title
  if (typeof title === 'object' && title !== null && 'value' in title) {
    return String((title as { value: unknown }).value)
  }
  return ''
}

export function extractFilename(path: string | undefined): string {
  if (!path) return ''
  return path.split('/').pop() ?? path
}

export function stripMarkdownExtension(filename: string): string {
  return filename.replace(/\.md$/, '')
}

export function extractStatePath(state: unknown): string {
  if (typeof state !== 'object' || state === null) return ''
  const s = state as Record<string, unknown>
  const val = s.path ?? s.file ?? s.at
  return typeof val === 'string' ? val : ''
}

const stylesRegistry = new Map<string, string>()

export function registerPluginStyle(pluginId: string, url: string): void {
  stylesRegistry.set(pluginId, url)
}

export function unregisterPluginStyle(pluginId: string): void {
  stylesRegistry.delete(pluginId)
}

export function getPluginStyle(pluginId: string): string | undefined {
  return stylesRegistry.get(pluginId)
}

export function clearPluginStyles(): void {
  stylesRegistry.clear()
}

export const PANE_LAYOUT_STYLE =
  ':host, .plugin-view-container { display: flex !important; flex-direction: column !important; width: 100% !important; height: 100% !important; min-width: 0 !important; min-height: 0 !important; flex: 1 1 0% !important; }'

export const PANE_HOST_STYLE: Record<string, string> = {
  display: 'flex',
  flexDirection: 'column',
  width: '100%',
  height: '100%',
  minWidth: '0',
  minHeight: '0',
  flex: '1 1 0%',
}

export function adoptHostStyles(shadowRoot: ShadowRoot): void {
  if (typeof document === 'undefined') return
  const hostStyles = document.querySelectorAll<HTMLStyleElement | HTMLLinkElement>(
    'head style:not([data-plugin-id]), head link[rel="stylesheet"]:not([data-plugin-id])',
  )
  for (const node of hostStyles) {
    shadowRoot.appendChild(node.cloneNode(true))
  }
}

export function attachPluginShadow(el: HTMLElement | null): ShadowRoot | null {
  if (!el) return null
  if (el.shadowRoot) return el.shadowRoot
  if (typeof el.attachShadow === 'function') {
    try {
      const shadow = el.attachShadow({ mode: 'open' })
      adoptHostStyles(shadow)
      return shadow
    } catch {
      return el.shadowRoot
    }
  }
  return null
}

export function renderFallback(pluginId: string, error: Error) {
  return h('div', { class: 'plugin-error-fallback p-4 text-red-500' }, [
    h('h3', { class: 'font-bold' }, `Plugin "${pluginId}" crashed`),
    h('pre', { class: 'text-xs mt-2' }, error.message),
  ])
}

export interface PaneContentDeps {
  readonly pluginId: string
  readonly component: Component | null
  readonly state: unknown
  readonly shadowRoot: ShadowRoot | null
  readonly containerClass: string
  readonly context?: PluginContext | undefined
}

export function renderPaneContent(deps: PaneContentDeps) {
  if (!deps.component) return null
  const stateObj =
    typeof deps.state === 'object' && deps.state !== null
      ? (deps.state as Record<string, unknown>)
      : {}
  const path = extractStatePath(deps.state)
  const child = h(deps.component, {
    ...stateObj,
    ...(path ? { path, filePath: path, file: path } : {}),
    state: deps.state,
    ...(deps.context ? { context: deps.context } : {}),
  })
  const styleUrl = getPluginStyle(deps.pluginId)
  const styleNode = styleUrl
    ? h('link', { rel: 'stylesheet', href: styleUrl, 'data-plugin-id': deps.pluginId })
    : null
  const layoutStyleNode = h('style', PANE_LAYOUT_STYLE)
  const inner = h(
    'div',
    {
      class: `${deps.containerClass} h-full w-full min-h-0 min-w-0 flex flex-col flex-1`,
      style: PANE_HOST_STYLE,
    },
    [child],
  )

  if (deps.shadowRoot) {
    return h(Teleport, { to: deps.shadowRoot }, [layoutStyleNode, styleNode, inner])
  }
  return [layoutStyleNode, styleNode, inner]
}

export function renderWrapper(opts: {
  wrapperClass: string
  pluginId: string
  hostEl: Ref<HTMLElement | null>
  content: VNodeChild
}) {
  return h(
    'div',
    {
      class: `${opts.wrapperClass} h-full w-full min-h-0 min-w-0 flex flex-col flex-1`,
      'data-plugin-id': opts.pluginId,
      ref: opts.hostEl,
      style: PANE_HOST_STYLE,
    },
    [opts.content],
  )
}
