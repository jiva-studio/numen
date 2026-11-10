/**
 * The entries of a search result that show a note, through the real pieces: the plugin host with the
 * slots of the manifests, the palette model, the runner and the handler table. Only the navigation
 * of a tab is a double.
 */
import { describe, expect, it, vi } from 'vitest'
import { SLOT, type CommandTarget as RunTarget } from '@numen/plugin-loader'
import { WORDS as windowWords } from '@/shared/ui/words'
import { createRealWindow, readManifest } from '@/testing/realCommands'
import { PALETTE_WORDS } from '../../words'
import { createNameItem, resolveChoice, type NameMatch } from './lookup'

const REEF: RunTarget = { path: 'Reef.md', title: 'Reef', isFolder: false, others: [], tab: '' }
const REVEAL = 'app:file.reveal'

const MANIFESTS = [readManifest('views-plex-graph'), readManifest('views-file-tree')]

const setup = () => {
  const navigateTab = vi.fn(async (_kind: string, _path: string) => undefined)
  return { navigateTab, ...createRealWindow({ manifests: MANIFESTS, navigateTab }) }
}

describe('the entries of a search result that show a note', () => {
  it('show the note found, not the one in front, in the tab of the kind they carry', async () => {
    const one = setup()
    const found: NameMatch = {
      path: 'Reef.md',
      title: 'Reef',
      heading: '',
      line: -1,
      at: [],
      type: 'note',
    }
    const entries = one.pluginHost.slots.getEntries(SLOT.paletteSearchResult, REEF)
    const row = createNameItem(found, { ...windowWords, ...PALETTE_WORDS }, entries)
    const actions = row.hit.actionIds.filter((id) => row.hit.actionCommands?.[id] === REVEAL)
    expect(actions).not.toStrictEqual([])

    for (const action of actions) {
      const choice = resolveChoice(row.hit, action)
      if (choice) one.used.runChoice(choice)
      const kind = row.hit.actionValues?.[action]?.['kind']
      await vi.waitFor(() => expect(one.navigateTab).toHaveBeenCalledWith(kind, 'Reef.md'))
    }
    expect(one.navigateTab).toHaveBeenCalledTimes(actions.length)
  })
})
