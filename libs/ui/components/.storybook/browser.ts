/**
 * The browsers a story is drawn in, wherever stories are run.
 *
 * Every package that runs stories takes its browsers from here, so each corpus
 * is drawn in the same engines with the same commands.
 */
import { accessSync, constants } from 'node:fs'
import { delimiter, join } from 'node:path'
import type { BrowserCommand } from 'vitest/node'
import { playwright } from '@vitest/browser-playwright'

/** A place on the page the browser drives the pointer to. */
interface Position {
  readonly x: number
  readonly y: number
}

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

const isExecutable = (path: string): boolean => {
  try {
    accessSync(path, constants.X_OK)
    return true
  } catch {
    return false
  }
}

const findExecutable = (names: readonly string[]): string | undefined =>
  (process.env['PATH'] ?? '')
    .split(delimiter)
    .filter(Boolean)
    .flatMap((directory) => names.map((name) => join(directory, name)))
    .find(isExecutable)

const getSystemChrome = (): string | undefined => {
  const explicit = process.env['CHROME_PATH']
  if (explicit) return explicit
  if (process.env['CI']) return undefined
  return findExecutable(CANDIDATES)
}

/**
 * The browser a story project runs in. The window is WebKit on a mac and on
 * Linux, and Chromium on Windows, so every story is rendered in both.
 */
export const getStoryBrowser = () => {
  const chrome = getSystemChrome()
  return {
    enabled: true,
    headless: true,
    provider: playwright(),
    // Storybook sets every story to 1200 by 900 unless the story names a viewport. A frame of
    // another size is resized at its first story, and that resize event can arrive after the
    // story's play has opened a menu, which closes on a resize.
    viewport: { width: 1200, height: 900 },
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
