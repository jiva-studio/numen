import { test, expect, type Locator, type Page } from '../../support/test.js'
import { observeFor } from '../../support/browser.js'
import { openWorkspace, runCommand } from '../../support/workspace.js'

/** What one frame, mutation or transition showed of the enabled switch and its card. */
interface Frame {
  /** Milliseconds since the click. */
  readonly elapsed: number
  /** What took the sample: a frame, a mutation or a transition event. */
  readonly source: string
  /** Whether the page drew no such switch. */
  readonly isAbsent: boolean
  /** The identity of the switch element, which changes where the page draws it anew. */
  readonly node: number
  readonly checked: string
  readonly state: string
  /** Where the thumb stands from the left edge of the switch. */
  readonly thumb: number
  /** The box of the header card and the top of the page that holds it. */
  readonly card: { x: number; y: number; width: number; height: number }
  readonly pageTop: number
}

declare global {
  interface Window {
    recordedFrames?: Frame[]
    stopFrames?: () => void
  }
}

interface Scenario {
  readonly name: string
  readonly id: string
}

const SCENARIOS: readonly Scenario[] = [
  { name: 'Index stats', id: 'md.numen.tools.index.stats' },
  { name: 'Update Notifier', id: 'md.numen.tools.updater' },
  { name: 'EPUB Book Reader', id: 'md.numen.readers.epub' },
  { name: 'Plex Graph', id: 'md.numen.views.plex.graph' },
]

/** The span the recording runs after a click, in milliseconds. */
const RECORD_MS = 8_000

/** Where the thumb stands from the left edge of the switch when it is off and on, in pixels. */
const THUMB_OFF = 2
const THUMB_ON = 18

/** The pixels of difference between two positions that count as the same one. */
const TOLERANCE = 0.5

const writeEnabled = async (page: Page, id: string, isOn: boolean): Promise<void> => {
  const answer = await page.request.post('/numen.v1.SettingsService/WriteSettings', {
    data: { settings: [{ at: ['plugins', id, 'enabled'], value: JSON.stringify(isOn) }] },
    headers: { 'Content-Type': 'application/json' },
  })
  expect(answer.ok()).toBe(true)
}

/**
 * How long the window waits for the answers of the core in the slow runs, in milliseconds. A plugin
 * that takes seconds to start holds the answer of the write of its switch, and a read that began
 * before it answers after it.
 */
const SLOW_WRITE_MS = 3_000
const SLOW_READ_MS = 1_500

/** Holds the answer of the write and of every read of the settings back, as the core gave it. */
const slowDown = async (page: Page): Promise<void> => {
  for (const [method, delay] of [
    ['WriteSettings', SLOW_WRITE_MS],
    ['GetSettings', SLOW_READ_MS],
  ] as const) {
    await page.route(`**/numen.v1.SettingsService/${method}`, async (route) => {
      const response = await route.fetch()
      await new Promise((resolve) => setTimeout(resolve, delay))
      await route.fulfill({ response })
    })
  }
}

/** How long reads show the old value after a write, in milliseconds. */
const LAG_MS = 4_000

/**
 * Has every read of the settings answer with the value the plugin had before the write for
 * `LAG_MS` after it, as a core does that applies a change to the plugin after it answers the write.
 */
const lagBehind = async (page: Page, id: string): Promise<void> => {
  let lag: { until: number; old: boolean } | null = null
  await page.route('**/numen.v1.SettingsService/WriteSettings', async (route) => {
    const body = route.request().postDataJSON() as { settings: { at: string[]; value: string }[] }
    const edit = body.settings.find((one) => one.at.join('.') === `plugins.${id}.enabled`)
    if (edit) lag = { until: Date.now() + LAG_MS, old: !(JSON.parse(edit.value) as boolean) }
    await route.continue()
  })
  await page.route('**/numen.v1.SettingsService/GetSettings', async (route) => {
    const response = await route.fetch()
    if (!lag || Date.now() > lag.until) {
      await route.fulfill({ response })
      return
    }
    const answer = (await response.json()) as { written: string }
    const settingsFile = JSON.parse(answer.written) as {
      plugins: Record<string, { enabled?: boolean }>
    }
    settingsFile.plugins[id] = { ...settingsFile.plugins[id], enabled: lag.old }
    await route.fulfill({ response, json: { ...answer, written: JSON.stringify(settingsFile) } })
  })
}

/** Waits until the card has kept its height for a second: the README of the plugin is shown. */
const waitForSettledCard = (page: Page): Promise<void> =>
  page.evaluate(
    () =>
      new Promise<void>((resolve) => {
        let height = -1
        let since = performance.now()
        const draw = (): void => {
          const now =
            document.querySelector('.plugin-detail__header')?.getBoundingClientRect().height ?? -1
          if (now !== height) {
            height = now
            since = performance.now()
          }
          if (performance.now() - since >= 1_000) resolve()
          else requestAnimationFrame(draw)
        }
        requestAnimationFrame(draw)
      }),
  )

const openPluginPage = async (page: Page, name: string): Promise<Locator> => {
  await runCommand(page, 'Settings', /^Settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  await settings.getByRole('button', { name: new RegExp(name) }).click()
  return settings
}

/** Starts sampling the switch labelled `label` on every frame, mutation and transition. */
const startRecording = async (page: Page, label: string): Promise<void> => {
  await page.evaluate((name) => {
    const frames: Frame[] = []
    const nodes = new WeakMap<Element, number>()
    let nodeCount = 0
    const startedAt = performance.now()
    let isStopped = false
    const sample = (source: string): void => {
      const root = document.querySelector(`[role="switch"][aria-label="${name}"]`)
      const header = document.querySelector('.plugin-detail__header')
      const page = document.querySelector('.plugin-detail')
      const elapsed = performance.now() - startedAt
      if (!root || !header || !page) {
        frames.push({
          elapsed,
          source,
          isAbsent: true,
          node: 0,
          checked: '',
          state: '',
          thumb: 0,
          card: { x: 0, y: 0, width: 0, height: 0 },
          pageTop: 0,
        })
        return
      }
      if (!nodes.has(root)) nodes.set(root, ++nodeCount)
      const rootBox = root.getBoundingClientRect()
      const thumbBox = (root.firstElementChild ?? root).getBoundingClientRect()
      const card = header.getBoundingClientRect()
      frames.push({
        elapsed,
        source,
        isAbsent: false,
        node: nodes.get(root) ?? 0,
        checked: root.getAttribute('aria-checked') ?? '',
        state: root.getAttribute('data-state') ?? '',
        thumb: thumbBox.x - rootBox.x,
        card: { x: card.x, y: card.y, width: card.width, height: card.height },
        pageTop: page.getBoundingClientRect().y,
      })
    }
    const observer = new MutationObserver(() => sample('mutation'))
    observer.observe(document.body, {
      subtree: true,
      childList: true,
      attributes: true,
      characterData: true,
    })
    for (const type of ['transitionrun', 'transitionstart', 'transitionend', 'transitioncancel']) {
      document.addEventListener(type, () => sample(type), true)
    }
    const draw = (): void => {
      if (isStopped) return
      sample('frame')
      requestAnimationFrame(draw)
    }
    requestAnimationFrame(draw)
    window.recordedFrames = frames
    window.stopFrames = () => {
      isStopped = true
      observer.disconnect()
    }
  }, label)
}

const stopRecording = (page: Page): Promise<Frame[]> =>
  page.evaluate(() => {
    window.stopFrames?.()
    return window.recordedFrames ?? []
  })

/** The values of a list in order, each once where it repeats directly. */
const getRuns = <T>(values: readonly T[]): T[] =>
  values.filter((one, index) => index === 0 || one !== values[index - 1])

const toggleAndRecord = async (page: Page, toggle: Locator, name: string): Promise<Frame[]> => {
  await startRecording(page, `Enabled ${name}`)
  await toggle.click()
  await observeFor(page, RECORD_MS)
  return stopRecording(page)
}

/** Whether the card of two frames differs in place or size. */
const isCardMoved = (a: Frame, b: Frame): boolean =>
  [
    Math.abs(a.card.y - b.card.y),
    Math.abs(a.card.x - b.card.x),
    Math.abs(a.card.width - b.card.width),
    Math.abs(a.card.height - b.card.height),
    Math.abs(a.pageTop - b.pageTop),
  ].some((distance) => distance > TOLERANCE)

/** The ways the thumb breaks its path from one end of the switch to the other. */
const findThumbJumps = (visible: readonly Frame[], isOn: boolean): string[] => {
  const problems: string[] = []
  let last = visible[0]?.thumb ?? 0
  for (const one of visible) {
    const step = isOn ? one.thumb - last : last - one.thumb
    if (step < -TOLERANCE) {
      problems.push(
        `thumb moved against its way at ${Math.round(one.elapsed)} ms (${last} to ${one.thumb})`,
      )
      break
    }
    last = one.thumb
  }
  const final = visible.at(-1)
  if (final && Math.abs(final.thumb - (isOn ? THUMB_ON : THUMB_OFF)) > TOLERANCE) {
    problems.push(`the thumb ended at ${final.thumb}`)
  }
  return problems
}

/** The ways the frames break what the switch promises, each with the frame that showed it. */
const findJumps = (frames: readonly Frame[], from: string, to: string): string[] => {
  const problems: string[] = []
  const visible = frames.filter((one) => !one.isAbsent)
  const absent = frames.find((one) => one.isAbsent)
  if (absent)
    problems.push(`no switch drawn at ${Math.round(absent.elapsed)} ms (${absent.source})`)
  const checked = getRuns(visible.map((one) => one.checked))
  if (checked.join('>') !== `${from}>${to}`) problems.push(`aria-checked ran ${checked.join('>')}`)
  const states = getRuns(visible.map((one) => one.state))
  if (states.length > 2) problems.push(`data-state ran ${states.join('>')}`)
  if (new Set(visible.map((one) => one.node)).size > 1) problems.push('the switch was drawn anew')
  problems.push(...findThumbJumps(visible, to === 'true'))
  const [first] = visible
  const moved = first && visible.find((one) => isCardMoved(one, first))
  if (first && moved) {
    problems.push(
      `the card moved at ${Math.round(moved.elapsed)} ms from ${JSON.stringify(first.card)} to ${JSON.stringify(moved.card)}`,
    )
  }
  return problems
}

test.describe('The enabled switch of a plugin, frame by frame', () => {
  test.describe.configure({ timeout: 180_000 })

  const MODES: readonly [string, (page: Page, id: string) => Promise<void>][] = [
    ['at the speed of the core', async () => undefined],
    ['with the core slow to answer', (page) => slowDown(page)],
    ['with reads that lag a write', lagBehind],
  ]

  for (const [mode, slowTheCore] of MODES) {
    for (const one of SCENARIOS) {
      test.describe(`${one.name} ${mode}`, () => {
        test.afterEach(async ({ page }) => {
          await writeEnabled(page, one.id, true)
        })

        test('changes once and moves its thumb one way, without moving the card', async ({
          page,
        }) => {
          await writeEnabled(page, one.id, true)
          await slowTheCore(page, one.id)
          await openWorkspace(page)
          const settings = await openPluginPage(page, one.name)
          const toggle = settings.getByRole('switch', { name: new RegExp(`Enabled ${one.name}`) })
          await expect(toggle).toHaveAttribute('aria-checked', 'true')
          await waitForSettledCard(page)

          const turnedOff = await toggleAndRecord(page, toggle, one.name)
          test.info().annotations.push({
            type: 'off',
            description: JSON.stringify(
              getRuns(turnedOff.map((frame) => `${frame.checked}@${Math.round(frame.elapsed)}`)),
            ),
          })
          expect(findJumps(turnedOff, 'true', 'false')).toEqual([])
          await expect(toggle).toHaveAttribute('aria-checked', 'false')
          await waitForSettledCard(page)

          const turnedOn = await toggleAndRecord(page, toggle, one.name)
          expect(findJumps(turnedOn, 'false', 'true')).toEqual([])
          await expect(toggle).toHaveAttribute('aria-checked', 'true')
        })
      })
    }
  }
})
