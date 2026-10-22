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

/**
 * The pointer put down at one place, carried to another and lifted, by the
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

/** Whether path is a file this process may run. */
const isExecutable = (path: string): boolean => {
  try {
    accessSync(path, constants.X_OK)
    return true
  } catch {
    return false
  }
}

/** The first of names that is an executable in a directory on PATH, in PATH order. */
const findExecutable = (names: readonly string[]): string | undefined =>
  (process.env['PATH'] ?? '')
    .split(delimiter)
    .filter(Boolean)
    .flatMap((directory) => names.map((name) => join(directory, name)))
    .find(isExecutable)

/**
 * A browser already on the machine; undefined uses Playwright's own.
 *
 * A run under `CI` takes the pinned build unless it was pointed at one, so the
 * version the stories are drawn in is the version the workflow fetched.
 */
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
