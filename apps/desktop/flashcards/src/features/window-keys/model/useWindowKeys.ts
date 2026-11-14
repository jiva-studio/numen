import { onMounted, onUnmounted, type Ref } from 'vue'
import { win } from '@numen/ui'
import type { Vault } from '@/entities/vault'
import type { WindowTabs } from '@/entities/tab'
import { createWindowKeys } from './createWindowKeys'

interface WindowKeysDeps {
  readonly tabs: Pick<WindowTabs, 'tab' | 'front' | 'openTab'>
  readonly vaults: Readonly<Ref<readonly Vault[]>>
}

/** Listen to the keys pressed in the window for as long as the component is mounted. */
export function useWindowKeys({ tabs, vaults }: WindowKeysDeps): void {
  const onKeyDown = createWindowKeys({
    getTab: () => tabs.tab.value,
    getVaults: () => (tabs.front.value ? vaults.value : []),
    openVault: tabs.openTab,
  })
  onMounted(() => win.addEventListener('keydown', onKeyDown))
  onUnmounted(() => win.removeEventListener('keydown', onKeyDown))
}
