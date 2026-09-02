/**
 * The picture of the window on this page, taken from the story that draws it.
 *
 * Storybook is started here and stopped again, so one command answers with the
 * two files the page asks for: the window in light, and the window in dark.
 */
import { spawn } from 'node:child_process'
import { accessSync, constants } from 'node:fs'
import { writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import sharp from 'sharp'

const HERE = dirname(fileURLToPath(import.meta.url))
const UI = join(HERE, '..', '..', '..', 'libs', 'ui')
const INTO = join(HERE, '..', 'src', 'assets')

/** A Storybook already running, for a machine that keeps one up. */
const GIVEN = process.env['STORYBOOK_PORT']
const PORT = GIVEN ? Number(GIVEN) : 6099
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
/** The window a person runs their cards in: standing, and not a wide window. */
const NARROW = { width: 760, height: 940 }

/**
 * What the page shows, one to a band and one over them all.
 *
 * `over` names what the pointer is left on: the picture is taken once whatever
 * a hand resting there brings out is all the way out. `size` is the window it
 * is drawn in, and the wide one where it names none.
 */
const SHOTS = [
  { name: 'filing', story: 'application-window--filing' },
  { name: 'writing', story: 'application-window--writing' },
  {
    name: 'mapping',
    story: 'application-window--mapping',
    over: '[aria-label="The second law, child"]',
  },
  { name: 'searching', story: 'application-window--searching' },
  { name: 'asking', story: 'application-window--asking' },
  { name: 'hearing', story: 'desktop-window--recording' },
  { name: 'transcribing', story: 'desktop-window--transcribed' },
  { name: 'recognising', story: 'desktop-window--recognised' },
  { name: 'decking', story: 'desktop-window--deck' },
  { name: 'cutting', story: 'desktop-window--stencil' },
  { name: 'pacing', story: 'desktop-window--preset' },
  { name: 'owing', story: 'flash-cards-window--owing', size: NARROW },
  { name: 'reviewing', story: 'flash-cards-window--reviewing', size: NARROW },
]
/** Pixels for one, so the picture holds up where it is drawn wide. */
const SCALE = 2.5
/** How long Storybook is given to answer before this gives up. */
const PATIENCE = 120_000
/** What the plex and the editor are given to settle once the page has loaded. */
const SETTLING = 2_000

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
      // A window of its own size for each picture: the page holds two shapes,
      // and one browser drawn at both would take the second at the first's.
      const page = await browser.newPage({
        viewport: shot.size ?? VIEWPORT,
        deviceScaleFactor: SCALE,
        colorScheme: theme,
      })
      await page.goto(
        `${base}/iframe.html?id=${shot.story}&viewMode=story&globals=theme:${theme}`,
        { waitUntil: 'networkidle' },
      )
      if (shot.over) await page.hover(shot.over)
      await page.waitForTimeout(SETTLING)
      const name = `window-${shot.name}-${theme}.webp`
      const taken = await sharp(await page.screenshot()).webp({ quality: QUALITY }).toBuffer()
      if (taken.length < DRAWN) throw new Error(`${name} is ${taken.length} bytes: the story had not drawn`)
      await writeFile(join(INTO, name), taken)
      console.log(name)
      await page.close()
    }
  }
  await browser.close()
} finally {
  if (storybook.pid) process.kill(-storybook.pid, 'SIGTERM')
}
