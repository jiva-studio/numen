/**
 * The drops plugins take: data dragged in from outside the window, handed to the first plugin that
 * registered one of its types, in the order the plugins registered them.
 */
import type { Disposable, DroppedData, DropRegistration, DropRegistry } from '@numen/plugin-sdk'

interface DeclaredDrop {
  readonly pluginId: string
  readonly registration: DropRegistration
}

export function createDropRegistry(): DropRegistry {
  const declared: DeclaredDrop[] = []

  const findDrop = (types: readonly string[]): DeclaredDrop | undefined =>
    declared.find((one) => one.registration.types.some((type) => types.includes(type)))

  const registerDropHandler = (pluginId: string, registration: DropRegistration): Disposable => {
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
    canDrop: (types) => findDrop(types) !== undefined,
    drop: async (folder: string, data: DroppedData) => {
      const found = findDrop(Object.keys(data).filter((type) => data[type] !== ''))
      return found ? found.registration.drop(folder, data) : []
    },
  }
}
