import type { Extension } from '@codemirror/state'
import { EditorView, ViewPlugin, tooltips } from '@codemirror/view'
import { doc } from '@numen/ui'
import { StyleModule } from 'style-mod'

const LAYER_ATTRIBUTE = 'data-numen-tooltip-layer'

function tooltipLayer(): HTMLElement {
  const existing = doc.querySelector<HTMLElement>(`[${LAYER_ATTRIBUTE}]`)
  if (existing) return existing
  const layer = doc.createElement('div')
  layer.setAttribute(LAYER_ATTRIBUTE, '')
  layer.style.cssText =
    'position:fixed;inset:0;overflow:hidden;pointer-events:none;z-index:2147483000'
  doc.body.appendChild(layer)
  return layer
}

function mountStyles(view: EditorView): void {
  StyleModule.mount(doc, view.state.facet(EditorView.styleModule).slice().reverse())
}

/**
 * Draws the tooltips of the editor in the page. The pane body sets `contain: strict`, which clips a
 * fixed element at the edge of the pane, so a list inside the editor is cut there. The editor styles
 * are mounted in the shadow root of the pane, so they are mounted in the page too, in the order the
 * editor mounts them, or a base rule wins over a rule of the theme. The tooltips sit in a fixed layer
 * clipped to the window, so one near the window edge cannot make the page scroll.
 */
export function pageTooltips(): Extension {
  return [
    tooltips({ parent: tooltipLayer() }),
    ViewPlugin.define((view) => {
      mountStyles(view)
      return { update: () => mountStyles(view) }
    }),
  ]
}
