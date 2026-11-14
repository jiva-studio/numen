import type { Ref } from 'vue'
import type { CommandTarget } from '@/shared/model/command'
import type { WindowTabs } from '@/entities/tab'
import type { useVaults } from '@/entities/vault'

/** What a command runs over when it names no target: the tab in front, in the vault the window shows. */
export function createTargetResolver(
  windowTabs: Pick<WindowTabs, 'groups' | 'tabs'>,
  vaults: Pick<ReturnType<typeof useVaults>, 'activeVault'>,
  health: { readonly failure: Readonly<Ref<string>> },
) {
  const getFront = () => {
    const front = windowTabs.groups.handle.front()
    const tab = front?.id ?? ''
    const on = front?.state && windowTabs.tabs.getTab(tab)?.kind.address?.getTarget?.(front.state)
    return { tab, kind: front?.kind ?? null, on }
  }
  return (): CommandTarget => {
    const { tab, kind, on } = getFront()
    const file = on?.file ?? ''
    return {
      tab,
      kind,
      path: on?.path ?? '',
      title: on?.title ?? '',
      file,
      vault: vaults.activeVault.value,
      isReady: !health.failure.value,
    }
  }
}
