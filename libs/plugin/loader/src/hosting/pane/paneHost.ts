/**
 * What a pane of a plugin view takes from the window: the path its state names, its layout, and the
 * stylesheets of the window.
 */
import type { Unvalidated } from '@numen/plugin-sdk'
import { createSheet } from '../../styles'

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
