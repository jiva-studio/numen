/**
 * How a plugin view is drawn inside its pane: its component, its stylesheet and the shadow root it
 * is contained in.
 */
import { defineAsyncComponent, h, Teleport, type Component, type Ref, type VNodeChild } from 'vue'
import type { PluginContext } from '@numen/plugin-sdk'
import { createSheet, getPluginStyle } from '@numen/plugin-host'

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

/** A constructed copy of a sheet of the window, read from its rules; nothing where it cannot be read. */
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
  if (copy) hostSheetCopies.set(sheet, copy)
  return copy
}

/**
 * The window's sheets as copies a shadow root adopts at once, or nothing where one of them cannot
 * be copied. A cloned link would draw the pane unstyled until it loads.
 */
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
  readonly state: unknown
  readonly shadowRoot: ShadowRoot | null
  readonly containerClass: string
  readonly context?: PluginContext | undefined
  /** Whether the shadow root adopted the plugin's sheet, so no link is drawn for it. */
  readonly hasAdoptedStyle?: boolean
}

export function renderPaneContent(deps: PaneContentDeps) {
  if (!deps.component) return null
  const stateObj =
    typeof deps.state === 'object' && deps.state !== null
      ? (deps.state as Record<string, unknown>)
      : {}
  const { title: _title, ...restState } = stateObj
  const path = extractStatePath(deps.state)
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
