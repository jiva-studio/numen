/**
 * Frontend entrypoint for the Plex Headings plugin: the headings of a note, listed under its node in the plex
 * while the pointer rests on it, drawn above everything else, beside the slot over the whole node that the pointer is read on.
 */
import { defineComponent, h, ref, type ComponentPublicInstance } from 'vue'
import { definePlugin, type Caught, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { readSettings } from './model/settings'
import { HOVER_WAIT, LEAVE_GRACE, sleep, type HeadingsSource } from './model/useHeadings'
import { useHeadingsView } from './model/useHeadingsView'
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
    },
    setup: (props) => {
      const trigger = ref<HTMLElement | null>(null)
      const { rows, layout, enter, leave, choose } = useHeadingsView(source, () => props, trigger)
      return () =>
        h(PlexHeadings, {
          ref: (instance) => {
            trigger.value = (instance as ComponentPublicInstance | null)?.$el ?? null
          },
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
    hoverWait: HOVER_WAIT,
    leaveGrace: LEAVE_GRACE,
  }
  plugin.register.slotView({ id: HEADINGS_VIEW, component: createHeadingsView(source) })
}

export default definePlugin({
  activate,
})
