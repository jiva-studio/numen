/** The value kept under a key; null where nothing is kept or the storage is closed, as in a private window. */
export function readStored(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

/** Keeps a value under a key; false where the storage is closed. */
export function writeStored(key: string, value: string): boolean {
  try {
    localStorage.setItem(key, value)
    return true
  } catch {
    return false
  }
}
