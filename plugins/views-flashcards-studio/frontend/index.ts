import { definePlugin, type PluginContext, type VaultFiles } from '@numen/plugin-sdk'
import { cards, presets } from './cards'
import { useDeckTabs } from './deck/model/useDeckTabs'
import { createCardModelTabs } from './card-model/kind'
import { usePresetTab } from './preset/kind'
import {
  createNotices,
  createReviewKind,
  createVaultCounts,
  createVaultSummaryView,
} from './review'

export { DECK } from './deck/kind'
export { CARD_MODEL } from './card-model/kind'
export { PRESET } from './preset/kind'
export { default as DeckTab } from './deck/ui/DeckTab.vue'
export { default as CardModelTab } from './card-model/ui/CardModelTab.vue'
export { default as PresetTab } from './preset/ui/PresetTab.vue'
export { useDeckTabs } from './deck/model/useDeckTabs'
export { createCardModelTabs } from './card-model/kind'
export { usePresetTab } from './preset/kind'
export type { DeckTabState } from './deck/types'
export type { CardModelTabState } from './card-model/types'
export type { PresetTabState } from './preset/types'

export { REVIEW } from './review'

export function activate(context: PluginContext): void {
  const { workspace, buffers, files } = context
  if (!workspace) {
    const { notices, closeTab, agent } = context
    if (!notices || !closeTab || !agent) {
      throw new Error(
        'the review needs the notices, the tabs and the agent of the window it is drawn in',
      )
    }
    const counts = createVaultCounts({ reportError: createNotices(notices).reportError })
    context.subscriptions.push({ dispose: counts.dispose })
    context.registerView(createVaultSummaryView(counts))
    context.registerTabKind(
      createReviewKind({
        counts,
        notices,
        agent,
        mountView: (capability, element, view) => context.mountView(capability, element, view),
        closeTab: (id) => closeTab(id),
      }),
    )
    return
  }
  if (!buffers) {
    throw new Error('the flashcards need the window their tabs are drawn in and its open files')
  }
  const closeTab = (id: string): void => context.closeTab?.(id)
  const decks = useDeckTabs(cards, presets, {
    buffers,
    closeTab,
    writeMessage: workspace.getMessageWriter('deck'),
  })
  const cardModels = createCardModelTabs(cards, {
    buffers,
    closeTab,
    writeMessage: workspace.getMessageWriter('card-model'),
  })
  const schedules = usePresetTab({
    core: presets,
    buffers,
    closeTab,
    writeMessage: workspace.getMessageWriter('preset'),
    today: () => workspace.getToday(),
  })
  context.registerTabKind(decks.kind)
  context.registerTabKind(cardModels.kind)
  context.registerTabKind(schedules.kind)
  context.registerCommand({
    id: 'preset.open',
    handler: async (note) => {
      await schedules.openPresetOf(note, (kind, path) => context.openTab?.(kind, path))
    },
  })

  if (files) {
    const commands: readonly (readonly [string, (folder: string) => Promise<unknown>])[] = [
      ['deck.new', (folder) => cards.createDeck('New deck', folder)],
      ['cardModel.new', (folder) => cards.createCardModel('New card model', folder, [])],
      ['preset.new', (folder) => presets.createPreset('New preset', folder)],
    ]
    for (const [id, create] of commands) {
      context.registerCommand({
        id,
        handler: async (note) => {
          await create(await getFolderOf(files, note.path))
        },
      })
    }
  }
}

/** The folder a file is made in for a row: the row where it is a folder, else the folder holding it. */
async function getFolderOf(files: VaultFiles, path: string): Promise<string> {
  if (path === '') return ''
  const slash = path.lastIndexOf('/')
  const parent = slash < 0 ? '' : path.slice(0, slash)
  const entries = await files.list(parent)
  return entries.some((one) => one.path === path && one.isFolder) ? path : parent
}

export default definePlugin({
  activate,
})
