/**
 * The deletion of a secret or of a whole group of secrets: what is asked, the confirmation and the
 * return of the focus to the button that opened it.
 */
import { computed, ref, watch } from 'vue'
import type { PluginInfo } from '@/entities/settings'
import { getDeletionIds, getDeletionText, type DeletionTarget } from '../lib/secretSections'

export function useSecretDeletion(
  deleteSecret: (id: string) => void,
  deleteSecretGroup: (group: string) => void,
  getPlugins: () => readonly PluginInfo[],
) {
  const target = ref<DeletionTarget | null>(null)
  const opener = ref<HTMLElement | null>(null)
  const text = computed(() => (target.value ? getDeletionText(target.value, getPlugins()) : ''))
  const isGroup = computed(() => target.value?.kind === 'group')

  /** Returns the focus to the button that opened the confirmation, if it is still there. */
  watch(
    target,
    (next) => {
      if (next !== null) return
      const button = opener.value
      opener.value = null
      if (button?.isConnected) button.focus()
    },
    { flush: 'post' },
  )

  function ask(next: DeletionTarget, button: HTMLElement): void {
    target.value = next
    opener.value = button
  }

  function confirm(): void {
    const current = target.value
    target.value = null
    if (current?.kind === 'group') deleteSecretGroup(current.section.group)
    else if (current) for (const id of getDeletionIds(current)) deleteSecret(id)
  }

  function cancel(): void {
    target.value = null
  }

  return { target, text, isGroup, ask, confirm, cancel }
}
