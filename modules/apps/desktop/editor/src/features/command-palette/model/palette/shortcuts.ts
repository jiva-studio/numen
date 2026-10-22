import { onMounted, onUnmounted } from 'vue'
import { isChord } from '../../lib/chords/chords'
import type { Commands } from './palette'
import type { SearchState } from '../search/search'

export function usePaletteShortcuts(search: SearchState, commands: Commands) {
  function onKeyDown(event: KeyboardEvent) {
    if (event.defaultPrevented || !isChord(event) || event.shiftKey) return
    const key = event.key.toLowerCase()
    if (key === 'k') {
      event.preventDefault()
      commands.setOpen(false)
      search.setOpen(!search.open.value)
      return
    }
    if (key === 'p') {
      event.preventDefault()
      search.setOpen(false)
      commands.setOpen(!commands.open.value)
    }
  }

  onMounted(() => globalThis.addEventListener('keydown', onKeyDown))
  onUnmounted(() => globalThis.removeEventListener('keydown', onKeyDown))
}
