import type { FrameRuntimeLoader } from '../lib/runtime'

/**
 * Opens an address in the system browser through the runtime, and in a new browsing context where the
 * page has none. The caller decides which schemes it opens. A failure of the runtime to open the
 * address rejects the returned promise.
 */
export async function openExternalUrl(href: string, load: FrameRuntimeLoader): Promise<void> {
  const open = (await load())?.Browser?.OpenURL
  if (open) {
    await open(href)
    return
  }
  window.open(href, '_blank')
}
