/**
 * The drops plugins take: data dragged in from outside the window, handed to the plugin that registered
 * one of its types; a type has one plugin, the first to register it. A plugin that declares a type
 * under `contributes.dropHandlers` and has not started takes the drop once the drop has started it.
 */
import type { Disposable, DroppedData, DropRegistration, DropRegistry } from '@numen/plugin-sdk'
import type { Manifest } from '@numen/core-client'
import type { ContributionMatch } from './loading'

interface DeclaredDrop {
  readonly pluginId: string
  readonly registration: DropRegistration
}

/** The plugins that declare drop types without having started. */
export interface DropRegistryDeps {
  readonly findPluginsOf?: (match: ContributionMatch) => readonly string[]
  readonly startPluginsOf?: (match: ContributionMatch) => Promise<void>
}

const declaresTypeOf =
  (types: readonly string[]) =>
  (manifest: Manifest): boolean =>
    (manifest.contributes?.dropHandlers ?? []).some((one) =>
      one.types.some((type) => types.includes(type)),
    )

export function createDropRegistry(deps: DropRegistryDeps = {}): DropRegistry {
  const declared: DeclaredDrop[] = []

  const findDrop = (types: readonly string[]): DeclaredDrop | undefined =>
    declared.find((one) => one.registration.types.some((type) => types.includes(type)))

  const registerDropHandler = (pluginId: string, registration: DropRegistration): Disposable => {
    for (const type of registration.types) {
      const holder = declared.find(
        (one) => one.pluginId !== pluginId && one.registration.types.includes(type),
      )
      if (holder) {
        throw new Error(
          `Plugin "${pluginId}" registers a drop handler for "${type}", which plugin "${holder.pluginId}" already takes`,
        )
      }
    }
    const drop: DeclaredDrop = { pluginId, registration }
    declared.push(drop)
    return {
      dispose: () => {
        const index = declared.indexOf(drop)
        if (index !== -1) declared.splice(index, 1)
      },
    }
  }

  return {
    registerDropHandler,
    canDrop: (types) =>
      findDrop(types) !== undefined ||
      (deps.findPluginsOf?.(declaresTypeOf(types)) ?? []).length > 0,
    drop: async (folder: string, data: DroppedData) => {
      const types = Object.keys(data).filter((type) => data[type] !== '')
      if (!findDrop(types)) await deps.startPluginsOf?.(declaresTypeOf(types))
      const found = findDrop(types)
      return found ? found.registration.drop(folder, data) : []
    },
  }
}
