import type { Extension } from '@codemirror/state'
import { EditorView, ViewPlugin, tooltips } from '@codemirror/view'
import { doc } from '@numen/ui'
import { StyleModule } from 'style-mod'

type Modules = readonly (StyleModule | readonly StyleModule[])[]

function readModules(view: EditorView): Modules {
  return view.state.facet(EditorView.styleModule).slice().reverse()
}

function isSame(a: Modules, b: Modules): boolean {
  return a.length === b.length && a.every((one, index) => one === b[index])
}

/**
 * Draws the tooltips of the editor in the page. The pane body sets `contain: strict`, which clips a
 * fixed element at the edge of the pane, so a list inside the editor is cut there. The editor styles
 * are mounted in the shadow root of the pane, so they are mounted in the page too, in the order the
 * editor mounts them, or a base rule wins over a rule of the theme.
 */
export function createPageTooltips(): Extension {
  return [
    tooltips({ parent: doc.body }),
    ViewPlugin.define((view) => {
      let mounted = readModules(view)
      StyleModule.mount(doc, mounted as StyleModule[])
      return {
        update: () => {
          const next = readModules(view)
          if (isSame(mounted, next)) return
          mounted = next
          StyleModule.mount(doc, next as StyleModule[])
        },
      }
    }),
  ]
}
