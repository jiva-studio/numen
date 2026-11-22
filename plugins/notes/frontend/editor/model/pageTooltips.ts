import type { Extension } from '@codemirror/state'
import { EditorView, ViewPlugin, tooltips } from '@codemirror/view'
import { doc } from '@numen/ui'
import { StyleModule } from 'style-mod'

function mountStyles(view: EditorView): void {
  StyleModule.mount(doc, view.state.facet(EditorView.styleModule).slice().reverse())
}

/**
 * Draws the tooltips of the editor in the page. A pane clips what it holds and makes it the containing
 * block of a fixed element, so a list drawn inside the editor is cut at the edge of the pane. The
 * styles of the editor are mounted in the shadow root of the pane, so they are mounted in the page too,
 * in the order the editor mounts them in, or a base rule would win over a rule of the theme.
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
