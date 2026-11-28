/**
 * Frontend entrypoint for the Plex Headings Inline plugin: the headings of a note, written as `# Heading` in the
 * slot under each node of the plex.
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
import { useInlineHeadings, type InlineSource } from './model/useInlineHeadings'
import InlineHeadings from './ui/InlineHeadings.vue'

/** The id the manifest declares the view under. */
const HEADINGS_VIEW = 'headings'

function createHeadingsView(source: InlineSource) {
  return defineComponent({
    name: 'NodeHeadings',
    props: { path: { type: String, required: true } },
    setup: (props) => {
      const finish = useSlotLoading()
      const { headings, isLoaded, choose } = useInlineHeadings(source, () => props.path)
      watch(
        isLoaded,
        (isDone) => {
          if (isDone) finish()
        },
        { immediate: true },
      )
      onScopeDispose(finish)
      return () => h(InlineHeadings, { headings: headings.value, onChoose: choose })
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
  const source: InlineSource = {
    notes,
    settings: readSettings(plugin),
    open: (path, line) => workspace.files.open(path, { placement: 'current', line }),
    onVaultChange: host.events.onVaultChange,
    reportFailure: (error: unknown) => notices.show(formatErrorMessage(error), 'error'),
  }
  plugin.register.slotView({ id: HEADINGS_VIEW, component: createHeadingsView(source) })
}

export default definePlugin({
  activate,
})
