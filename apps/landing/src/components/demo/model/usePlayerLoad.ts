import { onMounted } from 'vue'
import type { Player } from './player'

/** Loads the media of a player once the view is mounted. */
export function usePlayerLoad(getPlayer: () => Player, url: string): void {
  onMounted(() => getPlayer().load(url))
}
