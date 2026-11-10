/**
 * How a plugin view is rendered inside its pane: its component, its stylesheet
 * and the shadow root that contains it.
 */
import { defineAsyncComponent, h, Teleport, type Component, type Ref, type VNodeChild } from 'vue'
import type { PluginContext, Unvalidated } from '@numen/plugin-sdk'
import { createSheet, getPluginStyle } from '../styles'

type AsyncComponentRegistration = {
  type: 'async'
  loader: () => Promise<Component>
}

/** A component as a plugin registers it, or the ES module that exports it as `default`. */
export type PluginComponentSource = Component | { readonly default: Component }

/** A function that is no component: it has no component field and takes no props, so it loads one. */
function isLoaderFunction(comp: { readonly length: number }): boolean {
  return !('setup' in comp || 'render' in comp || 'props' in comp || '__name' in comp) && comp.length === 0
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
  return typeof comp === 'function'
    ? isLoaderFunction(comp)
    : isAsyncRegistration(comp)
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

/** The path a tab state names as `path`, `file` or `at`; the state is the plugin's own, so its shape is checked. */
export function parseStatePath(state: Unvalidated): string {
  if (typeof state !== 'object' || state === null) return ''
  const read = (key: 'path' | 'file' | 'at') => (key in state ? Reflect.get(state, key) : undefined)
  const val = read('path') ?? read('file') ?? read('at')
  return typeof val === 'string' ? val : ''
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

/** Constructed copies of the window's own sheets, by the sheet they copy. */
const hostSheetCopies = new WeakMap<CSSStyleSheet, CSSStyleSheet>()

function copyHostSheet(sheet: CSSStyleSheet): CSSStyleSheet | undefined {
  const cached = hostSheetCopies.get(sheet)
  if (cached) return cached
  let text: string
  try {
    text = [...sheet.cssRules].map((rule) => rule.cssText).join('\n')
    if (sheet.media.length > 0) text = `@media ${sheet.media.mediaText} {\n${text}\n}`
  } catch {
    return undefined
  }
  const copy = createSheet(text, sheet.href ?? undefined)
  // Stryker disable next-line ConditionalExpression: WeakMap lookup checks truthiness of cached sheet
  if (copy) hostSheetCopies.set(sheet, copy)
  return copy
}

function copyHostSheets(): CSSStyleSheet[] | undefined {
  const copies: CSSStyleSheet[] = []
  for (const sheet of [...document.styleSheets]) {
    if (sheet.disabled) continue
    const owner = sheet.ownerNode
    if (owner instanceof Element && owner.hasAttribute('data-plugin-id')) continue
    if (owner instanceof Element && !owner.closest('head')) continue
    const copy = copyHostSheet(sheet)
    if (!copy) return undefined
    copies.push(copy)
  }
  return copies
}

export function adoptHostStyles(shadowRoot: ShadowRoot): void {
  // Stryker disable next-line ConditionalExpression,EqualityOperator,StringLiteral: JSDOM always defines document
  if (typeof document === 'undefined') return
  const copies = 'adoptedStyleSheets' in shadowRoot ? copyHostSheets() : undefined
  if (copies) {
    shadowRoot.adoptedStyleSheets = [...copies, ...shadowRoot.adoptedStyleSheets]
    return
  }
  const hostStyles = document.querySelectorAll<HTMLStyleElement | HTMLLinkElement>(
    'head style:not([data-plugin-id]), head link[rel="stylesheet"]:not([data-plugin-id])',
  )
  for (const node of hostStyles) {
    shadowRoot.appendChild(node.cloneNode(true))
  }
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
    Object.entries(deps.state).filter(([key]) => key !== 'title'),
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
