/**
 * The pictures in the manual, taken from the stories the application is built
 * from.
 *
 * Nothing here is cropped by hand: every picture is a story, drawn at the size
 * it is drawn at, in light and in dark. The Storybooks the stories are drawn in
 * are started and stopped again, so one command answers with the whole set.
 */
import { accessSync, constants, readFileSync } from 'node:fs'
import { writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import sharp from 'sharp'

// By path and not by name, as the manual reads its story list.
import { openBooks } from '../../../tools/ci-stories/books.mjs'

import { SHOTS } from './shots.mjs'

const HERE = dirname(fileURLToPath(import.meta.url))
const INTO = join(HERE, '..', 'src', 'assets')
/** The palettes that ship inside the application, as the window wears them. */
const PRESETS = join(HERE, '..', '..', '..', 'libs', 'core', 'internal', 'adapter', 'theme', 'presets')

/** Pixels for one, so a picture holds up on a screen that has them. */
const SCALE = 2
/** How hard the pictures are pressed. Text stays clean at this. */
const QUALITY = 82
/** The fewest bytes a drawn window comes to. Under it, nothing was drawn. */
const DRAWN = 15_000
/** How long a story is given to draw before this gives up. */
const PATIENCE = 120_000
/** What a plex and an editor are given to settle once the page has loaded. */
const SETTLING = 2_000

/**
 * Whether a picture taken is the picture already on disk, pixel for pixel. The
 * encoder writes different bytes of the same window, so the file is compared by
 * what it draws.
 */
const same = async (at, taken) => {
  try {
    const [before, after] = await Promise.all([
      sharp(at).raw().toBuffer({ resolveWithObject: true }),
      sharp(taken).raw().toBuffer({ resolveWithObject: true }),
    ])
    return (
      before.info.width === after.info.width &&
      before.info.height === after.info.height &&
      before.data.equals(after.data)
    )
  } catch {
    // Nothing to compare with: there is no such picture yet, or it is unreadable.
    return false
  }
}

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
      const page = await browser.newPage({
        viewport: { width: shot.width, height: shot.height },
        deviceScaleFactor: SCALE,
        colorScheme: theme,
      })
      await page.goto(
        `${books.getBase(shot.story)}/iframe.html?id=${shot.story}&viewMode=story&globals=theme:${theme}`,
        { waitUntil: 'networkidle' },
      )
      // The screen is waited for, not timed: a story reached through a module
      // graph the server has not built yet takes longer than any pause, and a
      // story that will not load draws Storybook's error page instead.
      try {
        await page.waitForFunction(
          "!document.body.classList.contains('sb-show-errordisplay') && (document.querySelector('#storybook-root')?.children.length ?? 0) > 0",
          undefined,
          { timeout: PATIENCE },
        )
      } catch {
        throw new Error(`${shot.story} drew nothing this build could photograph`)
      }
      // A picture of a theme is the window wearing that theme: the palette
      // that ships with the application, spliced in as the window splices it.
      if (shot.preset) {
        await page.addStyleTag({ content: readFileSync(join(PRESETS, `${shot.preset}.css`), 'utf8') })
        await page.waitForTimeout(200)
      }

      if (shot.over) await page.hover(shot.over)

      await page.waitForTimeout(SETTLING)

      // WebP, because these are checked in: the same picture is a third of
      // the bytes, and what the build would convert it to anyway.
      const name = `${shot.name}-${theme}.webp`
      const taken = await sharp(await page.screenshot()).webp({ quality: QUALITY }).toBuffer()
      if (taken.length < (shot.drawn ?? DRAWN)) throw new Error(`${name} is ${taken.length} bytes: the story had not drawn`)
      const at = join(INTO, name)
      if (await same(at, taken)) {
        console.log(`${name} — the same picture`)
      } else {
        await writeFile(at, taken)
        console.log(name)
      }
      await page.close()
    }
  }
  await browser.close()
} finally {
  books.close()
}
