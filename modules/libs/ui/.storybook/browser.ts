/**
 * The browsers a story is drawn in, wherever stories are run.
 *
 * Every package that runs stories takes its browsers from here, so each corpus
 * is drawn in the same engines with the same commands.
 */
import { accessSync, constants } from 'node:fs'
import { join } from 'node:path'
import type { BrowserCommand } from 'vitest/node'
import { playwright } from '@vitest/browser-playwright'

/** A place on the page the browser drives the pointer to. */
interface Position {
  readonly x: number
  readonly y: number
}

/**
 * The pointer put down at one place, carried to another and lifted, by the
 * browser itself. What a drag leaves behind — a selection, a capture — is then
 * the browser's own.
 */
const sweep: BrowserCommand<[from: Position, to: Position]> = async ({ page }, from, to) => {
  await page.mouse.move(from.x, from.y)
  await page.mouse.down()
  await page.mouse.move(to.x, to.y)
  await page.mouse.up()
}

declare module 'vitest/browser' {
  interface BrowserCommands {
    sweep: (from: Position, to: Position) => Promise<void>
  }
}

const CANDIDATES = ['google-chrome-stable', 'google-chrome', 'chromium', 'chromium-browser']

/**
 * A browser already on the machine; undefined uses Playwright's own.
 *
 * A run under `CI` takes the pinned build unless it was pointed at one, so the
 * version the stories are drawn in is the version the workflow fetched.
 */
function getSystemChrome(): string | undefined {
  const explicit = process.env['CHROME_PATH']
  if (explicit) return explicit
  if (process.env['CI']) return undefined

  for (const directory of (process.env['PATH'] ?? '').split(':')) {
    if (!directory) continue
    for (const name of CANDIDATES) {
      const candidate = join(directory, name)
      try {
        accessSync(candidate, constants.X_OK)
        return candidate
      } catch {
        continue
      }
    }
  }
  return undefined
}

/**
 * The browser a story project runs in. The window is WebKit on a mac and on
 * Linux, and Chromium on Windows, so every story is rendered in both.
 */
export function getStoryBrowser() {
  const chrome = getSystemChrome()
  return {
    enabled: true,
    headless: true,
    provider: playwright(),
    instances: [
      {
        browser: 'chromium' as const,
        ...(chrome ? { provider: playwright({ launchOptions: { executablePath: chrome } }) } : {}),
      },
      { browser: 'webkit' as const },
    ],
    commands: { sweep },
  }
}
