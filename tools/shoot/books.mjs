/**
 * The Storybooks a camera's shots are drawn in, started for the run and
 * stopped again: the components' own, the shell's, the editor's or the
 * plugins'. Only the ones named are started. `STORYBOOK_PORT`, `STORYBOOK_SHELL_PORT`,
 * `STORYBOOK_EDITOR_PORT` and `STORYBOOK_PLUGINS_PORT` name one already
 * running instead.
 */
import { join } from 'node:path'
import { root } from '../shared/modules.mjs'
import { spawnTool } from '../shared/tools.mjs'
import { BOOKS } from '../ci-stories/stories.mjs'

/** The port each Storybook is started on, and the variable naming one already up. */
const PORTS = {
  ui: { given: 'STORYBOOK_PORT', port: 6098 },
  shell: { given: 'STORYBOOK_SHELL_PORT', port: 6097 },
  editor: { given: 'STORYBOOK_EDITOR_PORT', port: 6095 },
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

const stopAll = (started) => {
  for (const one of started) {
    try {
      process.kill(-one.pid, 'SIGTERM')
    } catch {
      // The Storybook has already exited.
    }
  }
}

/**
 * Every Storybook named, answering. `getBase` is the address a Storybook is
 * served at; `close` stops what was started here.
 */
export const openBooks = async (books) => {
  const started = []
  const bases = new Map()
  try {
    for (const book of new Set(books)) {
      const { given, port: own } = PORTS[book]
      const port = process.env[given] ? Number(process.env[given]) : own
      if (!process.env[given]) {
        started.push(
          // npx runs in the nearest package folder, which is not the Storybook's own for the plugins'.
          spawnTool(
            'npx',
            [
              'storybook',
              'dev',
              '-p',
              String(port),
              '-c',
              join(root, BOOKS[book], '.storybook'),
              '--no-open',
              '--quiet',
            ],
            {
              cwd: join(root, BOOKS[book]),
              detached: true,
              stdio: 'ignore',
            },
          ),
        )
      }
      const base = `http://localhost:${port}`
      await waitForAnswer(`${base}/iframe.html`)
      bases.set(book, base)
    }
  } catch (error) {
    stopAll(started)
    throw error
  }

  return {
    getBase: (book) => bases.get(book),
    close: () => {
      stopAll(started)
    },
  }
}
