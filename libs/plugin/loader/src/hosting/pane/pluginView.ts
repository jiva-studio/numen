/**
 * How a plugin view is rendered inside its pane: its component, its stylesheet
 * and the shadow root that contains it.
 */
import {
  defineAsyncComponent,
  h,
  isRef,
  Teleport,
  type Component,
  type Ref,
  type VNodeChild,
} from 'vue'
import type { PluginContext, Unvalidated } from '@numen/plugin-sdk'
import { getPluginStyle } from '../../styles'
import { adoptHostStyles, parseStatePath, PANE_HOST_STYLE, PANE_LAYOUT_STYLE } from './paneHost'

type AsyncComponentRegistration = {
  type: 'async'
  loader: () => Promise<Component>
}

/** A component as a plugin registers it, or the ES module that exports it as `default`. */
export type PluginComponentSource = Component | { readonly default: Component }

function isLoaderFunction(comp: { readonly length: number }): boolean {
  return (
    !('setup' in comp || 'render' in comp || 'props' in comp || '__name' in comp) &&
    comp.length === 0
  )
}

function isAsyncRegistration(comp: Unvalidated): comp is AsyncComponentRegistration {
  return (
    typeof comp === 'object' &&
    comp !== null &&
    'type' in comp &&
    comp.type === 'async' &&
    'loader' in comp &&
    typeof comp.loader === 'function'
  )
}

function isAsyncComponent(
  comp: Unvalidated,
): comp is (() => Promise<Component>) | AsyncComponentRegistration {
  return typeof comp === 'function' ? isLoaderFunction(comp) : isAsyncRegistration(comp)
}

/**
 * The component a registration names. A plugin bundle hands it over as the component, as a module
 * that exports it as `default`, or as a loader that resolves to it; none of these is typed alike, so
 * the shape is checked here.
 */
export function parsePluginComponent(rawComponent: Unvalidated): Component | null {
  if (!rawComponent) return null
  const comp =
    typeof rawComponent === 'object' && 'default' in rawComponent
      ? rawComponent.default
      : rawComponent
  if (!comp) return null
  if (isAsyncComponent(comp)) {
    return defineAsyncComponent(comp)
  }
  return comp as Component
}

/** The text of a title that a tab state keeps as a string or as a ref, and nothing for any other value. */
export function parseTitle(title: Unvalidated): string {
  if (typeof title === 'string') return title
  if (typeof title === 'object' && title !== null && 'value' in title) {
    return String(title.value)
  }
  return ''
}

export function extractFilename(path: string | undefined): string {
  if (!path) return ''
  // Stryker disable next-line LogicalOperator,MethodExpression: split always returns at least one element
  return path.split('/').pop() ?? path
}

export function stripMarkdownExtension(filename: string): string {
  return filename.replace(/\.md$/, '')
}

export function attachPluginRoot(el: HTMLElement | null): ShadowRoot | null {
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
  readonly state: object
  readonly shadowRoot: ShadowRoot | null
  readonly containerClass: string
  readonly context?: PluginContext | undefined
  readonly hasAdoptedStyle?: boolean
}

export function renderPaneContent(deps: PaneContentDeps) {
  if (!deps.component) return null
  const restState = Object.fromEntries(
    Object.entries(deps.state).filter(([key, value]) => key !== 'title' && !isRef(value)),
  )
  const path = parseStatePath(deps.state)
  const child = h(deps.component, {
    ...restState,
    ...(path ? { path, filePath: path, file: path } : {}),
    state: deps.state,
    ...(deps.context ? { context: deps.context } : {}),
  })
  const styleUrl = deps.hasAdoptedStyle ? undefined : getPluginStyle(deps.pluginId)
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
