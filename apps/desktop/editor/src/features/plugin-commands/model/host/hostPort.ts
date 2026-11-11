import { shallowRef, watch } from 'vue'
import type { CommandsHost } from '@numen/plugin-loader'
import type { CommandTarget, VaultRef } from '@/entities/command'

/**
 * The command operations the plugin host delegates to the window's command palettes once the
 * palettes are composed with the host.
 */
interface HostCommandPort {
  readonly commands: CommandsHost
  readonly isPaletteOpen: () => boolean
  readonly runInvocation: (id: string, target: CommandTarget, name: string) => Promise<void>
  readonly openVault: (vault: VaultRef) => Promise<void>
}

/**
 * The port between the plugin host and the command palettes: each needs the other, so the host
 * reads through this port and the palettes connect it once they are made.
 */
export interface HostPort extends HostCommandPort {
  readonly connect: (port: HostCommandPort) => void
}

export function createHostPort(): HostPort {
  const connected = shallowRef<HostCommandPort | undefined>(undefined)
  const read = (): HostCommandPort => {
    if (connected.value === undefined) {
      throw new Error('The plugin host was asked before the window commands were connected')
    }
    return connected.value
  }
  const commands: CommandsHost = {
    getIds: () => connected.value?.commands.getIds() ?? [],
    watch: (handler) => {
      let inner: { dispose(): void } | undefined
      const stop = watch(
        connected,
        (port) => {
          inner?.dispose()
          inner = port?.commands.watch(handler)
        },
        { immediate: true },
      )
      return {
        dispose: () => {
          stop()
          inner?.dispose()
        },
      }
    },
    listItems: (id, target) => read().commands.listItems(id, target),
    run: (id, target, itemId, values) => read().commands.run(id, target, itemId, values),
  }
  return {
    commands,
    isPaletteOpen: () => read().isPaletteOpen(),
    runInvocation: (id, target, name) => read().runInvocation(id, target, name),
    openVault: (vault) => read().openVault(vault),
    connect: (port: HostCommandPort): void => {
      if (connected.value !== undefined) {
        throw new Error('The plugin host command port is connected already')
      }
      connected.value = port
    },
  }
}
