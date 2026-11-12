/**
 * The keybindings the enabled plugins declare, held by the plugin host from the moment the core lists
 * them, so a key works before its plugin starts.
 */
import { watch, type Ref } from 'vue'
import type { PluginHost } from '@numen/plugin-loader'
import type { DeclaredKeybinding } from '@/entities/plugin'

type KeybindingHost = Pick<
  PluginHost['keybindings'],
  'registerKeybindings' | 'unregisterKeybindings'
>

const groupByPlugin = (
  keybindings: readonly DeclaredKeybinding[],
): ReadonlyMap<string, readonly DeclaredKeybinding[]> => {
  const groups = new Map<string, DeclaredKeybinding[]>()
  for (const one of keybindings)
    groups.set(one.pluginId, [...(groups.get(one.pluginId) ?? []), one])
  return groups
}

/**
 * Registers the declared keybindings of each plugin with the host as the list changes, and again as
 * the plugins start or stop, which makes the host drop the keybindings of a plugin that stops.
 */
export function syncKeybindings(
  host: KeybindingHost,
  keybindings: Ref<readonly DeclaredKeybinding[] | undefined>,
  plugins?: Ref<readonly object[]>,
): void {
  let owners: ReadonlySet<string> = new Set()
  const register = () => {
    const groups = groupByPlugin(keybindings.value ?? [])
    for (const owner of owners) if (!groups.has(owner)) host.unregisterKeybindings(owner)
    for (const [owner, list] of groups) host.registerKeybindings(owner, list)
    owners = new Set(groups.keys())
  }
  watch([keybindings, () => plugins?.value], register, { flush: 'post', immediate: true })
}
