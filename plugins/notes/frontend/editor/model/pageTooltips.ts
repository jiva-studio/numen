import type { Extension } from '@codemirror/state'
import { EditorView, ViewPlugin, tooltips } from '@codemirror/view'
import { doc } from '@numen/ui'
import { StyleModule } from 'style-mod'

function mountStyles(view: EditorView): void {
  StyleModule.mount(doc, view.state.facet(EditorView.styleModule).slice().reverse())
}

/**
 * Draws the tooltips of the editor in the page. The pane body sets `contain: strict`, which clips a
 * fixed element at the edge of the pane, so a list inside the editor is cut there. The editor styles
 * are mounted in the shadow root of the pane, so they are mounted in the page too, in the order the
 * editor mounts them, or a base rule wins over a rule of the theme.
 */
export function pageTooltips(): Extension {
  return [
    tooltips({ parent: doc.body }),
    ViewPlugin.define((view) => {
      mountStyles(view)
      return { update: () => mountStyles(view) }
    }),
  ]
}
