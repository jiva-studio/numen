/**
 * The Storybooks a camera's shots are drawn in, started for the run and
 * stopped again.
 *
 * Each shot is served by the Storybook its story is declared in: the
 * components' own or the plugins'. Only the ones a list of shots names are
 * started. `STORYBOOK_PORT` and `STORYBOOK_PLUGINS_PORT` name one already
 * running instead.
 */
import { spawn } from 'node:child_process'
import { join } from 'node:path'
import { root } from '../shared/modules.mjs'
import { BOOKS, stories } from './stories.mjs'

/** The port each Storybook is started on, and the variable naming one already up. */
const PORTS = {
  ui: { given: 'STORYBOOK_PORT', port: 6098 },
  plugins: { given: 'STORYBOOK_PLUGINS_PORT', port: 6096 },
}

/** How long a Storybook is given to answer before the run gives up. */
const PATIENCE = 120_000

const waitForAnswer = async (url) => {
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

/**
 * Every Storybook these shots are drawn in, answering. `getBase` is the
 * address a shot's story is served at; `close` stops what was started here.
 */
export const openBooks = async (shots) => {
  const declared = await stories()
  const bookOf = (story) => {
    const book = declared.get(story)
    if (!book || book === 'both') throw new Error(`${story} is drawn in no one Storybook`)
    return book
  }

  const started = []
  const bases = new Map()
  try {
    for (const book of new Set(shots.map((shot) => bookOf(shot.story)))) {
      const { given, port: own } = PORTS[book]
      const port = process.env[given] ? Number(process.env[given]) : own
      if (!process.env[given]) {
        started.push(
          spawn('npx', ['storybook', 'dev', '-p', String(port), '--no-open', '--quiet'], {
            cwd: join(root, BOOKS[book]),
            detached: true,
            stdio: 'ignore',
          }),
        )
      }
      const base = `http://localhost:${port}`
      await waitForAnswer(`${base}/iframe.html`)
      bases.set(book, base)
    }
  } catch (error) {
    for (const one of started) process.kill(-one.pid, 'SIGTERM')
    throw error
  }

  return {
    getBase: (story) => bases.get(bookOf(story)),
    close: () => {
      for (const one of started) process.kill(-one.pid, 'SIGTERM')
    },
  }
}
