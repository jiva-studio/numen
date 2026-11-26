/**
 * Takes the screenshots the stories declare, in light and in dark, and writes them as WebP.
 *
 * Usage: node tools/shoot/shoot.mjs [--app <docs|landing>]... [--plugin <folder>]... [--story <id>]...
 *
 * With no option every declared shot is taken. `--app` and `--plugin` keep the shots written into
 * that app or plugin, `--story` the shots of that story. The Storybooks the shots are drawn in are
 * started for the run and stopped again; a picture whose pixels are unchanged is not rewritten.
 */
import { accessSync, constants } from 'node:fs'
import { mkdir, writeFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import { parseArgs } from 'node:util'
import { chromium } from 'playwright'
import sharp from 'sharp'
import { readPreset } from '../ci-stories/presets.mjs'
import { root } from '../shared/modules.mjs'
import { openBooks } from './books.mjs'
import { APPS, getFiles, getPluginFolder, readShots } from './shots.mjs'

const SCALE = 2.5
const QUALITY = 82
/** The fewest bytes a drawn window comes to. Under it, nothing was drawn. */
const DRAWN = 15_000
const PATIENCE = 120_000
/** What a plex and an editor are given to settle once the story has drawn. */
const SETTLING = 2_000
const THEMES = ['light', 'dark']

/**
 * Whether a picture taken is the picture already on disk, pixel for pixel. The encoder writes
 * different bytes for the same image.
 */
const isSamePicture = async (at, taken) => {
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
    return false
  }
}

const CANDIDATES = ['google-chrome-stable', 'google-chrome', 'chromium', 'chromium-browser']

/** The Chrome `CHROME_PATH` names, else one on PATH, else one Playwright downloaded, else none. */
const findChrome = () => {
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

const takeShot = async (browser, base, shot, theme, at) => {
  const page = await browser.newPage({
    viewport: { width: shot.width, height: shot.height },
    deviceScaleFactor: SCALE,
    colorScheme: theme,
  })
  try {
    await page.goto(
      `${base}/iframe.html?id=${shot.story}&viewMode=story&globals=theme:${theme}`,
      { waitUntil: 'networkidle' },
    )
    // A story reached through a module graph the server has not built yet takes longer than any
    // pause, and a story that fails draws Storybook's error page.
    try {
      await page.waitForFunction(
        "!document.body.classList.contains('sb-show-errordisplay') && (document.querySelector('#storybook-root')?.children.length ?? 0) > 0",
        undefined,
        { timeout: PATIENCE },
      )
    } catch {
      throw new Error(`${shot.story} drew nothing`)
    }
    if (shot.preset) {
      await page.addStyleTag({ content: readPreset(shot.preset) })
      await page.waitForTimeout(200)
    }
    if (shot.over) await page.hover(shot.over)
    await page.waitForTimeout(SETTLING)
    const taken = await sharp(await page.screenshot()).webp({ quality: QUALITY }).toBuffer()
    if (taken.length < (shot.drawn ?? DRAWN)) {
      throw new Error(`${at} is ${taken.length} bytes: ${shot.story} had not drawn`)
    }
    const path = join(root, at)
    if (await isSamePicture(path, taken)) {
      process.stdout.write(`${at} — the same picture\n`)
      return
    }
    await mkdir(dirname(path), { recursive: true })
    await writeFile(path, taken)
    process.stdout.write(`${at}\n`)
  } finally {
    await page.close()
  }
}

const { values } = parseArgs({
  options: {
    app: { type: 'string', multiple: true, default: [] },
    plugin: { type: 'string', multiple: true, default: [] },
    story: { type: 'string', multiple: true, default: [] },
  },
})

const { shots: declared, faults } = await readShots()
if (faults.length > 0) {
  process.stderr.write(`shoot: ${faults.join('\n')}\n`)
  process.exit(1)
}
const folders = [...values.app.map((app) => APPS[app]), ...values.plugin.map(getPluginFolder)]
const shots = declared.filter(
  (shot) =>
    (folders.length === 0 || folders.includes(shot.into)) &&
    (values.story.length === 0 || values.story.includes(shot.story)),
)
if (shots.length === 0) {
  process.stderr.write('shoot: no declared shot matches the options\n')
  process.exit(1)
}

const books = await openBooks(shots.map((shot) => shot.book))
const failed = []
try {
  const chrome = findChrome()
  const browser = await chromium.launch(chrome ? { executablePath: chrome } : {})
  for (const shot of shots) {
    const files = getFiles(shot)
    for (const [index, theme] of THEMES.entries()) {
      try {
        await takeShot(browser, books.getBase(shot.book), shot, theme, files[index])
      } catch (error) {
        failed.push(`${files[index]}: ${error instanceof Error ? error.message : String(error)}`)
      }
    }
  }
  await browser.close()
} finally {
  books.close()
}
if (failed.length > 0) {
  process.stderr.write(`shoot: ${failed.length} pictures failed\n${failed.join('\n')}\n`)
  process.exit(1)
}
