/**
 * Frontend entrypoint for the Related Notes plugin: the names of the notes related to the notes of the plex,
 * drawn in the slots over and under each of its nodes.
 */
import { defineComponent, h } from 'vue'
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { readSettings } from './model/settings'
import { useNames, type NamesSource } from './model/useNames'
import type { NamesSide } from './types'
import RelatedNames from './ui/RelatedNames.vue'

/** The ids the manifest declares the two views under. */
const PARENTS_VIEW = 'parents'
const RELATED_VIEW = 'related'

function createNamesView(source: NamesSource, side: NamesSide) {
  return defineComponent({
    name: side === 'above' ? 'ParentNames' : 'RelatedNotesNames',
    props: { path: { type: String, required: true } },
    setup: (props) => {
      const { names, choose } = useNames(source, side, () => props.path)
      return () => h(RelatedNames, { names: names.value, onChoose: choose })
    },
  })
}

export function activate(plugin: PluginContext, host: HostServices): void {
  const { notes } = host.vault
  const { commands, notices } = host.ui
  if (!notes || !commands) {
    throw new Error(
      'the names of related notes need the notes of the vault and the commands of the window',
    )
  }
  const source: NamesSource = {
    notes,
    commands,
    settings: readSettings(plugin),
    reportFailure: (error) =>
      notices.show(error instanceof Error ? error.message : String(error), 'error'),
  }
  plugin.register.slotView({ id: PARENTS_VIEW, component: createNamesView(source, 'above') })
  plugin.register.slotView({ id: RELATED_VIEW, component: createNamesView(source, 'below') })
}

export default definePlugin({
  activate,
})
