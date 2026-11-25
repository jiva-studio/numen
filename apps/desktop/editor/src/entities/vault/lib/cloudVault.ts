/**
 * The vaults kept in the cloud, and what the window shows of those that are not on this device yet.
 */

/** One vault of the cloud. A non-empty `vaultId` names the vault of this device it is connected to. */
export interface CloudVault {
  readonly remoteId: string
  readonly name: string
  readonly sizeBytes: number
  readonly vaultId: string
}

/** Where a cloud vault would be put: the folder, the full path and whether that path is in use. */
export interface VaultPlace {
  readonly folder: string
  readonly path: string
  readonly isTaken: boolean
}

const ROW_PREFIX = 'cloud:'

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB'] as const

const BASE = 1024

/** The id a row of a list takes for a cloud vault, kept apart from the ids of local vaults. */
export const getCloudRowId = (remoteId: string): string => `${ROW_PREFIX}${remoteId}`

/** The id of the cloud vault a row stands for, and nothing for a row of another kind. */
export const findCloudRemoteId = (rowId: string): string =>
  rowId.startsWith(ROW_PREFIX) ? rowId.slice(ROW_PREFIX.length) : ''

/** The cloud vaults that are not on this device. A connected one is shown once, as its local vault. */
export const selectNewCloudVaults = (list: readonly CloudVault[]): readonly CloudVault[] =>
  list.filter((one) => one.vaultId === '')

/** The size of a vault as a person reads it: "1.9 GB". */
export const formatVaultSize = (bytes: number): string => {
  let value = Math.max(bytes, 0)
  let unit = 0
  while (value >= BASE && unit < UNITS.length - 1) {
    value /= BASE
    unit += 1
  }
  const shown = unit === 0 ? Math.round(value) : Number(value.toFixed(value >= 10 ? 0 : 1))
  return `${shown} ${UNITS[unit]}`
}

/** The path of a vault called `name` inside `folder`, written with the separator the folder uses. */
export const joinFolder = (folder: string, name: string): string => {
  const separator = folder.includes('\\') && !folder.includes('/') ? '\\' : '/'
  return folder.endsWith(separator) ? `${folder}${name}` : `${folder}${separator}${name}`
}
