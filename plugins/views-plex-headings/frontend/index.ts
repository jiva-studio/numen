/**
 * Frontend entrypoint for the Plex Headings plugin: the headings of a note, listed under its node in the plex
 * while the pointer rests on it, drawn in the slot over the whole node.
 */
import { defineComponent, h, type PropType } from 'vue'
import { definePlugin, type Caught, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { readSettings } from './model/settings'
import { sleep, type HeadingsSource } from './model/useHeadings'
import { useHeadingsView } from './model/useHeadingsView'
import type { Size } from './types'
import PlexHeadings from './ui/PlexHeadings.vue'

/** The id the manifest declares the view under. */
const HEADINGS_VIEW = 'headings'

function createHeadingsView(source: HeadingsSource) {
  return defineComponent({
    name: 'PlexNodeHeadings',
    props: {
      path: { type: String, required: true },
      x: { type: Number, required: true },
      y: { type: Number, required: true },
      width: { type: Number, required: true },
      height: { type: Number, required: true },
      scale: { type: Number, required: true },
      viewport: { type: Object as PropType<Size>, required: true },
    },
    setup: (props) => {
      const { rows, layout, enter, leave, choose } = useHeadingsView(source, () => props)
      return () =>
        h(PlexHeadings, {
          rows: rows.value,
          layout: layout.value,
          onEnter: enter,
          onLeave: leave,
          onChoose: choose,
        })
    },
  })
}

export function activate(plugin: PluginContext, host: HostServices): void {
  const { notes } = host.vault
  const { workspace, notices } = host.ui
  if (!notes || !workspace) {
    throw new Error(
      'the headings of a note need the notes of the vault and the window they open in',
    )
  }
  const source: HeadingsSource = {
    notes,
    settings: readSettings(plugin),
    open: (path, line) => workspace.files.open(path, { placement: 'current', line }),
    reportFailure: (error: Caught) =>
      notices.show(error instanceof Error ? error.message : String(error), 'error'),
    wait: sleep,
  }
  plugin.register.slotView({ id: HEADINGS_VIEW, component: createHeadingsView(source) })
}

export default definePlugin({
  activate,
})
