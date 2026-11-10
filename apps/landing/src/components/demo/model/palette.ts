import { computed, ref } from 'vue'
import {
  NAMING_LABELS,
  getCommandGroups,
  getNamingGroups,
  getSearchGroups,
  type NamingRole,
} from '../lib/palette'
import type { Vault } from './vault'

type Panel = 'search' | 'commands' | 'naming'

const SEARCH_START = 'entrop'

/** The palette: which panel is open, what is typed in it, and the rows it shows. */
export function createPalette(vault: Vault) {
  const panel = ref<Panel | null>(null)
  const typed = ref(SEARCH_START)
  const naming = ref<{ role: NamingRole; path: string } | null>(null)

  const groups = computed(() => {
    if (panel.value === 'naming') return getNamingGroups(typed.value)
    if (panel.value === 'commands') return getCommandGroups(typed.value)
    return getSearchGroups(Object.values(vault.notes), typed.value)
  })
  const placeholder = computed(() => {
    if (panel.value === 'naming') return 'Name'
    return panel.value === 'commands' ? 'Type a command' : 'Search the vault'
  })
  const crumb = computed(() => {
    if (panel.value === 'naming' && naming.value) return NAMING_LABELS[naming.value.role]
    return panel.value === 'commands' ? 'Command' : ''
  })
  const step = computed(() => {
    if (panel.value === 'naming' && naming.value) return `1:${naming.value.role}:naming`
    return panel.value === 'commands' ? '0::commands' : ''
  })

  function toggle(next: 'search' | 'commands') {
    if (panel.value === next) {
      panel.value = null
      return
    }
    typed.value = next === 'search' ? SEARCH_START : ''
    panel.value = next
  }

  return {
    panel,
    typed,
    naming,
    groups,
    placeholder,
    crumb,
    step,
    toggle,
    openNaming(role: NamingRole, path: string) {
      naming.value = { role, path }
      typed.value = role === 'title' ? (vault.getNote(path)?.title ?? '') : ''
      panel.value = 'naming'
    },
    dismiss() {
      panel.value = null
      naming.value = null
    },
  }
}

export type Palette = ReturnType<typeof createPalette>
