/**
 * Frontend entrypoint for the Plex Related Names plugin: the names of the notes related to the notes of the plex,
 * drawn in the slots over and under each of its nodes.
 */
import { defineComponent, h, onScopeDispose, watch } from 'vue'
import {
  definePlugin,
  useSlotLoading,
  type HostServices,
  type PluginContext,
  formatErrorMessage,
} from '@numen/plugin-sdk'
import { readSettings } from './model/settings'
import { createNotesReader } from './model/notesReader'
import { createShownPaths } from './model/shownPaths'
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
      const finish = useSlotLoading()
      const { names, isLoaded, choose } = useNames(source, side, () => props.path)
      watch(
        isLoaded,
        (isDone) => {
          if (isDone) finish()
        },
        { immediate: true },
      )
      onScopeDispose(finish)
      return () =>
        h(RelatedNames, {
          names: names.value,
          side,
          showArrows: source.settings.showRelationArrows.value,
          onChoose: choose,
        })
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
  const settings = readSettings(plugin)
  const reportFailure = (error: unknown) => notices.show(formatErrorMessage(error), 'error')
  const reader = createNotesReader({ notes, events: host.events, reportFailure })
  plugin.identity.subscriptions.push({ dispose: reader.dispose })
  const source: NamesSource = {
    reader,
    commands,
    settings,
    shown: createShownPaths(),
    reportFailure,
  }
  plugin.register.slotView({ id: PARENTS_VIEW, component: createNamesView(source, 'above') })
  plugin.register.slotView({ id: RELATED_VIEW, component: createNamesView(source, 'below') })
}

export default definePlugin({
  activate,
})
