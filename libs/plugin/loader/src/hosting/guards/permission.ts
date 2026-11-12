import type { Manifest } from '@numen/core-client'

export interface PermissionTarget {
  readonly pluginId: string
  readonly manifest?: Manifest | { permissions?: readonly string[] } | undefined
}

/** Throws where a plugin lacks a permission, naming the operation that needed it. */
export type RequirePermission = (permission: string, operation: string) => void

export function checkPermission(
  target: PermissionTarget,
  permission: string,
  operation: string,
): void {
  if (!(target.manifest?.permissions?.includes(permission) ?? false)) {
    throw new Error(
      `Plugin "${target.pluginId}" lacks required permission "${permission}" for ${operation}`,
    )
  }
}
