/**
 * The pictures in the manual, taken from the stories the application is built
 * from.
 *
 * Nothing here is cropped by hand: every picture is a story, drawn at the size
 * it is drawn at, in light and in dark. Storybook is started and stopped again,
 * so one command answers with the whole set.
 */
import { spawn } from 'node:child_process'
import { accessSync, constants, readFileSync } from 'node:fs'
import { writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import sharp from 'sharp'

const HERE = dirname(fileURLToPath(import.meta.url))
const UI = join(HERE, '..', '..', '..', 'libs', 'ui')
const INTO = join(HERE, '..', 'src', 'assets')
/** The palettes that ship inside the application, as the window wears them. */
const PRESETS = join(HERE, '..', '..', '..', 'libs', 'core', 'internal', 'adapter', 'theme', 'presets')

/** A Storybook already running, for a machine that keeps one up. */
const GIVEN = process.env['STORYBOOK_PORT']
const PORT = GIVEN ? Number(GIVEN) : 6098
/** Pixels for one, so a picture holds up on a screen that has them. */
const SCALE = 2
/** How hard the pictures are pressed. Text stays clean at this. */
const QUALITY = 82
/** The fewest bytes a drawn window comes to. Under it, nothing was drawn. */
const DRAWN = 15_000
/** How long Storybook is given to answer before this gives up. */
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

/**
 * What is taken, and from which story.
 *
 * A window is drawn narrower than the column it lands in, so the application's
 * own text is larger on the page than it is on screen and can be read at a
 * glance. What stands alone is drawn at about the size it stands at.
 *
 * `over` names what the pointer is left on: the picture is taken once whatever
 * a hand resting there brings out is all the way out.
 */
export const SHOTS = [
  { name: 'window', story: 'application-window--map', width: 1180, height: 740 },
  { name: 'theme-dracula', story: 'application-window--map', width: 1180, height: 740, preset: 'dracula' },
  { name: 'theme-solarized', story: 'application-window--map', width: 1180, height: 740, preset: 'solarized' },
  { name: 'searching', story: 'application-window--searching', width: 1180, height: 740 },
  { name: 'asking', story: 'application-window--asking', width: 1180, height: 740 },
  { name: 'plex', story: 'application-window--mapping', width: 1180, height: 740 },
  {
    name: 'parts',
    story: 'application-window--hanging',
    width: 1180,
    height: 740,
    over: '.plex__node--focus',
  },
  { name: 'commands', story: 'application-window--commanding', width: 1180, height: 740 },
  { name: 'writing', story: 'application-window--writing', width: 1180, height: 740 },
  { name: 'table', story: 'application-window--tabling', width: 1180, height: 740 },
  { name: 'reader', story: 'application-window--reading', width: 1180, height: 740 },
  { name: 'files', story: 'application-window--filing', width: 1180, height: 740 },
  { name: 'settings', story: 'desktop-window--settings', width: 1180, height: 740 },
  { name: 'recording', story: 'desktop-window--recording', width: 1180, height: 740 },
  { name: 'transcribe', story: 'desktop-window--transcribed', width: 1180, height: 740 },
  { name: 'recognise', story: 'desktop-window--recognised', width: 1180, height: 740 },
  { name: 'deck', story: 'desktop-window--deck', width: 1180, height: 740 },
  { name: 'stencil', story: 'desktop-window--stencil', width: 1180, height: 740 },
  { name: 'preset', story: 'desktop-window--preset', width: 1180, height: 740 },
  // The window a person runs their cards in, which is not a wide window.
  { name: 'decks', story: 'flash-cards-window--owing', width: 760, height: 540 },
  { name: 'sitting', story: 'flash-cards-window--reviewing', width: 760, height: 540 },
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
  const storybook = GIVEN
    ? { pid: 0 }
    : spawn('npx', ['storybook', 'dev', '-p', String(PORT), '--no-open', '--quiet'], {
        cwd: UI,
        detached: true,
        stdio: 'ignore',
      })

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
        // The screen is waited for, not timed: a story reached through a module
        // graph the server has not built yet takes longer than any pause, and a
        // story that will not load draws Storybook's error page instead.
        try {
          await page.waitForFunction(
            () =>
              !document.body.classList.contains('sb-show-errordisplay') &&
              (document.querySelector('#storybook-root')?.children.length ?? 0) > 0,
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
        if (taken.length < DRAWN) throw new Error(`${name} is ${taken.length} bytes: the story had not drawn`)
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
    if (storybook.pid) process.kill(-storybook.pid, 'SIGTERM')
  }
}
