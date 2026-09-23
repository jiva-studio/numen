/**
 * What the phone's page may load, held against what it does load.
 *
 * The policy is in `index.html`, so what is checked here is the one the WebView
 * is served. Nothing here runs on a phone: what a WebView makes of the policy is
 * answered by a device and by nothing else.
 */
import { readFileSync, readdirSync } from 'node:fs'
import { createRequire } from 'node:module'
import { join } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import NoteSheet from './note/NoteSheet.vue'
import type { Core } from './core'

/** The origin the platform serves the page from: `androidScheme` and nothing. */
const PAGE = 'http://localhost'

/** This module, on disk. The runner serves it, so its own address is one. */
const here = new URL('..', import.meta.url).pathname.replace(/^\/@fs/, '')

/** The policy the page carries, directive by directive. */
const policy: Record<string, string[]> = Object.fromEntries(
  (
    /http-equiv="Content-Security-Policy"\s+content="([^"]+)"/.exec(
      readFileSync(join(here, 'index.html'), 'utf8'),
    )?.[1] ?? ''
  )
    .split(';')
    .map((one) => one.trim().split(/\s+/))
    .map(([named, ...sources]) => [named, sources]),
)

/**
 * Whether a directive lets an address be asked for.
 *
 * Only the source forms this policy writes are understood: a keyword, a scheme
 * on its own, and a host with a port that may be any.
 */
function isAllowed(directive: string, address: string): boolean {
  const sources = policy[directive] ?? policy['default-src'] ?? []
  return sources.some((source) => {
    if (source === "'self'") return new URL(address, PAGE).origin === PAGE
    if (source.startsWith("'")) return false
    if (/^[a-z][a-z0-9+.-]*:$/.test(source)) return address.startsWith(source)
    const named = /^(?:([a-z]+):\/\/)?([^:/]+)(?::(\d+|\*))?$/.exec(source)
    if (!named) return false
    const [, scheme, host, port] = named
    const at = new URL(address, PAGE)
    return (
      (!scheme || at.protocol === `${scheme}:`) &&
      at.hostname === host &&
      (port === '*' || (port ?? '') === at.port)
    )
  })
}

/** Every file the page is built from. Its tests are not built into it. */
function readSources(from: string): string[] {
  return readdirSync(from, { withFileTypes: true }).flatMap((entry) => {
    const path = join(from, entry.name)
    if (entry.isDirectory()) return readSources(path)
    if (/\.test\.ts$/.test(entry.name)) return []
    return /\.(ts|vue)$/.test(entry.name) ? [readFileSync(path, 'utf8')] : []
  })
}

/** The stylesheets `main.ts` loads, in the order it loads them. */
const STYLESHEETS = [
  '@ionic/vue/css/core.css',
  '@ionic/vue/css/normalize.css',
  '@ionic/vue/css/structure.css',
  '@ionic/vue/css/typography.css',
  '@numen/ui/styles.css',
]

/** Resolves a name this package imports the way the build resolves it: by the manifest. */
const from = createRequire(join(here, 'package.json'))

afterEach(() => {
  vi.unstubAllGlobals()
  vi.resetModules()
  document.body.innerHTML = ''
})

describe('the policy the page carries', () => {
  it('is the whole of what the page may load', () => {
    expect(policy).toStrictEqual({
      'default-src': ["'self'"],
      'img-src': ["'self'", 'data:'],
      'style-src': ["'self'", "'unsafe-inline'"],
      'font-src': ["'self'"],
      'connect-src': ["'self'", 'http://127.0.0.1:*'],
      'object-src': ["'none'"],
      'base-uri': ["'none'"],
      'form-action': ["'none'"],
    })
  })

  it('writes no directive a page may not give itself', () => {
    // A policy in the document is refused these three, and saying them anyway
    // buys nothing but a warning in the log.
    for (const named of ['frame-ancestors', 'report-uri', 'sandbox']) {
      expect(policy).not.toHaveProperty(named)
    }
  })

  it('submits a form nowhere', () => {
    // Android does not ask before a POST navigation, so the policy is what
    // holds a form the page never wrote from carrying the window away.
    expect(isAllowed('form-action', 'https://evil.example/submitted')).toBe(false)
  })
})

describe('what the page asks for', () => {
  it('reaches the core where the policy allows', async () => {
    const asked: string[] = []
    vi.stubGlobal('fetch', async (input: unknown) => {
      asked.push(input instanceof Request ? input.url : String(input))
      throw new Error('this test serves nothing')
    })
    vi.doMock('@capacitor/core', () => ({
      registerPlugin: () => ({ start: async () => ({ port: 45123, dir: '/data' }) }),
    }))

    const { reach } = await import('./core')
    const core = await reach()
    await expect(core.notes.getNeighbourhood({ path: 'Physics.md' })).rejects.toThrow()

    expect(asked).not.toStrictEqual([])
    for (const address of asked) expect(isAllowed('connect-src', address)).toBe(true)
  })

  it('asks for nothing from a stylesheet', () => {
    for (const sheet of STYLESHEETS) {
      const css = readFileSync(from.resolve(sheet), 'utf8')
      expect([
        sheet,
        [...css.matchAll(/url\(\s*['"]?([^'")]*)/g)].map((at) => at[1]),
      ]).toStrictEqual([sheet, []])
    }
  })

  it('names no host but the core in what it is built from', () => {
    const named = readSources(join(here, 'src')).flatMap((source) =>
      [...source.matchAll(/https?:\/\/([^:/'"`\s$]+)/g)].map((at) => at[1]),
    )
    expect([...new Set(named)]).toStrictEqual(['127.0.0.1'])
  })
})

describe('a picture written into a note', () => {
  it('is drawn at the address the note gave, which the policy refuses', async () => {
    const core = {
      notes: {
        readNote: vi.fn().mockResolvedValue({
          body: 'A note synced from elsewhere.\n\n![](https://evil.example/pixel.png)\n',
          at: undefined,
        }),
      },
    } as unknown as Core

    const sheet = mount(NoteSheet, { props: { core, path: 'Synced.md' }, attachTo: document.body })
    await flushPromises()

    // The editor draws a picture where one was written, at the address written,
    // and a note came from wherever it was synced from.
    const drawn = [...document.querySelectorAll('.cm-picture img')].map(
      (each) => (each as HTMLImageElement).src,
    )
    expect(drawn).toStrictEqual(['https://evil.example/pixel.png'])
    for (const address of drawn) expect(isAllowed('img-src', address)).toBe(false)

    sheet.unmount()
  })
})
