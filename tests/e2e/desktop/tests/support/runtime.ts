/**
 * A stand-in of the Wails runtime the window serves at /wails/runtime.js. It names the platform,
 * which the frame and the plugins read, and records the calls of the window buttons.
 */
import type { Page } from 'playwright/test'

export type Platform = 'windows' | 'mac'

const createRuntime = (platform: Platform): string => `
const calls = (globalThis.__frameCalls ??= [])
export const Window = {
  Minimise: async () => { calls.push('Minimise') },
  ToggleMaximise: async () => { calls.push('ToggleMaximise') },
  Close: async () => { calls.push('Close') },
  IsMaximised: async () => false,
  IsFullscreen: async () => false,
}
export const System = {
  IsMac: () => ${platform === 'mac'},
  IsWindows: () => ${platform === 'windows'},
  IsLinux: () => false,
}
`

export const serveRuntime = async (page: Page, platform: Platform): Promise<void> => {
  await page.route('**/wails/runtime.js', (route) =>
    route.fulfill({ contentType: 'text/javascript', body: createRuntime(platform) }),
  )
}

/** The calls of the window buttons the runtime has recorded, in order. */
export const readFrameCalls = (page: Page): Promise<string[]> =>
  page.evaluate(() => [...((globalThis as { __frameCalls?: string[] }).__frameCalls ?? [])])
