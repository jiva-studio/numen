import { getAssetUrl, type Result } from '@numen/plugin-sdk'

/** The text of the file at a vault path, served by the window; the error is why it could not be read. */
export async function readTextAsset(path: string): Promise<Result<string, string>> {
  const response = await fetch(getAssetUrl(path))
  if (!response.ok) return { ok: false, error: `${path}: ${response.status}` }
  return { ok: true, value: await response.text() }
}
