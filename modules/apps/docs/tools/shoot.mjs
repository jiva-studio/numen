/**
 * The pictures in the manual, taken from the stories the application is built
 * from.
 *
 * Nothing here is cropped by hand: every picture is a story, drawn at the size
 * it is drawn at, in light and in dark. Storybook is started and stopped again,
 * so one command answers with the whole set.
 */
import { spawn } from 'node:child_process'
import { accessSync, constants } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import sharp from 'sharp'

const HERE = dirname(fileURLToPath(import.meta.url))
const UI = join(HERE, '..', '..', '..', 'libs', 'ui')
const INTO = join(HERE, '..', 'src', 'assets')

const PORT = 6098
/** Pixels for one, so a picture holds up on a screen that has them. */
const SCALE = 2
/** How hard the pictures are pressed. Text stays clean at this. */
const QUALITY = 82
/** How long Storybook is given to answer before this gives up. */
const PATIENCE = 120_000
/** What a plex and an editor are given to settle once the page has loaded. */
const SETTLING = 2_000

/**
 * What is taken, and from which story.
 *
 * A window is drawn narrower than the column it lands in, so the application's
 * own text is larger on the page than it is on screen and can be read at a
 * glance. What stands alone is drawn at about the size it stands at.
 */
export const SHOTS = [
  { name: 'window', story: 'application-window--map', width: 1180, height: 740 },
  { name: 'searching', story: 'application-window--searching', width: 1180, height: 740 },
  { name: 'asking', story: 'application-window--asking', width: 1180, height: 740 },
  { name: 'plex', story: 'plex-plex--titled-lines', width: 1040, height: 700 },
  { name: 'commands', story: 'generic-palette--key-hints', width: 900, height: 620 },
  { name: 'writing', story: 'text-editor--playground', width: 900, height: 560 },
  { name: 'table', story: 'text-editor--table', width: 900, height: 520 },
  { name: 'reader', story: 'reading-reader--lit-over', width: 960, height: 700 },
  { name: 'files', story: 'tree--several', width: 760, height: 560 },
]

const CANDIDATES = ['google-chrome-stable', 'google-chrome', 'chromium', 'chromium-browser']

/** A browser already on the machine; undefined uses Playwright's own. */
const systemChrome = () => {
  const explicit = process.env['CHROME_PATH']
  if (explicit) return explicit

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

const answering = async (url) => {
  const until = Date.now() + PATIENCE
  while (Date.now() < until) {
    try {
      const answer = await fetch(url)
      if (answer.ok) return
    } catch {
      // Not up yet.
    }
    await new Promise((wake) => setTimeout(wake, 500))
  }
  throw new Error(`Storybook did not answer at ${url}`)
}

// Imported by the check that the stories are still there, which asks for the
// list and nothing else.
if (process.argv[1] !== fileURLToPath(import.meta.url)) {
  // Nothing to do: the list above is the export.
} else {
  const storybook = spawn(
    'npx',
    ['storybook', 'dev', '-p', String(PORT), '--no-open', '--quiet'],
    { cwd: UI, detached: true, stdio: 'ignore' },
  )

  try {
    const base = `http://localhost:${PORT}`
    await answering(`${base}/iframe.html`)

    const chrome = systemChrome()
    const browser = await chromium.launch(chrome ? { executablePath: chrome } : {})
    for (const theme of ['light', 'dark']) {
      for (const shot of SHOTS) {
        const page = await browser.newPage({
          viewport: { width: shot.width, height: shot.height },
          deviceScaleFactor: SCALE,
          colorScheme: theme,
        })
        await page.goto(
          `${base}/iframe.html?id=${shot.story}&viewMode=story&globals=theme:${theme}`,
          { waitUntil: 'networkidle' },
        )
        await page.waitForTimeout(SETTLING)

        // WebP, because these are checked in: the same picture is a third of
        // the bytes, and what the build would convert it to anyway.
        const name = `${shot.name}-${theme}.webp`
        await sharp(await page.screenshot()).webp({ quality: QUALITY }).toFile(join(INTO, name))
        console.log(name)
        await page.close()
      }
    }
    await browser.close()
  } finally {
    if (storybook.pid) process.kill(-storybook.pid, 'SIGTERM')
  }
}
