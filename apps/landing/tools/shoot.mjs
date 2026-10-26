/**
 * The picture of the window on this page, taken from the story that draws it.
 *
 * The Storybooks the stories are drawn in are started here and stopped again, so
 * one command answers with every file the page asks for, in light and in dark.
 */
import { accessSync, constants } from 'node:fs'
import { writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import sharp from 'sharp'

// By path and not by name, as the story check reads its list.
import { openBooks } from '../../../tools/ci-stories/books.mjs'

import { SHOTS } from './shots.mjs'

const HERE = dirname(fileURLToPath(import.meta.url))
const INTO = join(HERE, '..', 'src', 'assets')

/** How hard the pictures are pressed. The build presses them again. */
const QUALITY = 90
/** The fewest bytes a drawn window comes to. Under it, nothing was drawn. */
const DRAWN = 20_000
/*
 * A window narrower than the column it is drawn in, so the application's own
 * text lands on the page larger than it stands on screen and can be read at a
 * glance.
 */
const VIEWPORT = { width: 1180, height: 740 }
/** Pixels for one, so the picture holds up where it is drawn wide. */
const SCALE = 2.5
/** What the plex and the editor are given to settle once the page has loaded. */
const SETTLING = 2_000

const CANDIDATES = ['google-chrome-stable', 'google-chrome', 'chromium', 'chromium-browser']

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
  const cache = join(process.env['HOME'] ?? '', '.cache', 'ms-playwright')
  for (const name of [
    'chromium-1243/chrome-linux64/chrome',
    'chromium-1234/chrome-linux64/chrome',
    'chromium-1228/chrome-linux64/chrome',
  ]) {
    const candidate = join(cache, name)
    try {
      accessSync(candidate, constants.X_OK)
      return candidate
    } catch {
      continue
    }
  }
  return undefined
}

const books = await openBooks(SHOTS)

try {
  const chrome = systemChrome()
  const browser = await chromium.launch(chrome ? { executablePath: chrome } : {})
  for (const theme of ['light', 'dark']) {
    for (const shot of SHOTS) {
      // A window of its own size for each picture: the page holds two shapes,
      // and one browser drawn at both would take the second at the first's.
      const page = await browser.newPage({
        viewport: shot.size ?? VIEWPORT,
        deviceScaleFactor: SCALE,
        colorScheme: theme,
      })
      await page.goto(
        `${books.getBase(shot.story)}/iframe.html?id=${shot.story}&viewMode=story&globals=theme:${theme}`,
        { waitUntil: 'networkidle' },
      )
      if (shot.over) await page.hover(shot.over)
      await page.waitForTimeout(SETTLING)
      const name = `window-${shot.name}-${theme}.webp`
      const taken = await sharp(await page.screenshot()).webp({ quality: QUALITY }).toBuffer()
      if (taken.length < (shot.drawn ?? DRAWN)) throw new Error(`${name} is ${taken.length} bytes: the story had not drawn`)
      await writeFile(join(INTO, name), taken)
      console.log(name)
      await page.close()
    }
  }
  await browser.close()
} finally {
  books.close()
}
