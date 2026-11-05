/**
 * What a control of the page System shows while a change of its setting goes through the window:
 * the write, its answer, the setting-changed event and the reads that follow arrive in turn, and
 * the control shows the value the person set from the first moment on. Every value here is invented.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount, type VueWrapper } from '@vue/test-utils'
import { Select, Slider } from '@numen/ui'
import SystemDetail from '@/pages/settings/ui/SystemDetail.vue'
import SettingRow from '@/pages/settings/ui/setting-row/SettingRow.vue'
import { useWindowTabs } from '@/entities/tab'
import type { PluginSettingInfo, SettingEdit } from '@/entities/settings'
import type { SettingsPort } from '@/app/ports/settings'
import type { MessageLog } from '@/shared/notices/messages'
import { WORDS } from '@/shared/words'
import { useSettings } from './useSettings'

type Tree = Record<string, Record<string, unknown>>

const file = vi.hoisted(() => ({ current: {} as Record<string, unknown> }))

vi.mock('@/entities/settings', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/entities/settings')>()),
  themes: {
    getAppearance: async () => {
      const appearance = ((file.current['app'] as Tree)['appearance'] ?? {}) as Record<
        string,
        unknown
      >
      return {
        themes: [],
        applied: appearance['theme'] as string,
        mode: appearance['mode'] as string,
        sizes: { interfaceScale: appearance['interfaceScale'], textScale: 1 },
        bounds: { interfaceScale: { min: 0.8, max: 2 }, textScale: { min: 0.8, max: 2 } },
      }
    },
    readTheme: async (name: string) => `:root { --theme: "${name}"; }`,
    writeAppearance: async () => '',
    watchThemes: async function* () {
      yield* []
    },
  },
}))

const THEME_CHOICES = [
  { value: 'preset:numen', title: 'Numen', detail: '' },
  { value: 'plugin:md.example.themes/dusk', title: 'Dusk', detail: '' },
]
const MODE_CHOICES = [
  { value: 'system', title: 'System', detail: '' },
  { value: 'light', title: 'Light', detail: '' },
  { value: 'dark', title: 'Dark', detail: '' },
]
const PROFILE_CHOICES = [
  { value: 'eco', title: 'Eco', detail: '' },
  { value: 'balanced', title: 'Balanced', detail: '' },
  { value: 'maximum', title: 'Maximum', detail: '' },
]

const DECLARED = [
  { key: 'appearance.theme', title: 'Theme', type: 'enum', default: 'preset:numen', c: THEME_CHOICES },
  { key: 'appearance.mode', title: 'Light or dark', type: 'enum', default: 'system', c: MODE_CHOICES },
  { key: 'appearance.interfaceScale', title: 'Interface size', type: 'range', default: 1 },
  { key: 'performance.profile', title: 'Performance profile', type: 'enum', default: 'balanced', c: PROFILE_CHOICES },
] as const

const getValueAt = (tree: unknown, key: string): unknown =>
  key.split('.').reduce<unknown>((one, step) => (one as Record<string, unknown> | undefined)?.[step], tree)

/** The declaration of the shell as the core answers it: each setting with the value in force. */
const declare = (): readonly PluginSettingInfo[] =>
  DECLARED.map((one) => {
    const choices = 'c' in one ? one.c : undefined
    return {
      key: one.key,
      type: one.type,
      title: one.title,
      description: '',
      default: one.default,
      isShown: true,
      value: getValueAt(file.current['app'], one.key) ?? one.default,
      ...(choices ? { options: choices, activeOptions: choices } : {}),
      ...(one.type === 'range' ? { min: 0.8, max: 2, step: 0.1 } : {}),
    } as PluginSettingInfo
  })

const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T

const createFeed = () => {
  const pending: ((batch: readonly SettingEdit[]) => void)[] = []
  return {
    open: (): AsyncIterable<readonly SettingEdit[]> => ({
      [Symbol.asyncIterator]: () => ({
        next: async () => ({
          done: false,
          value: await new Promise<readonly SettingEdit[]>((resolve) => pending.push(resolve)),
        }),
      }),
    }),
    push: (batch: readonly SettingEdit[]) => pending.shift()?.(batch),
  }
}

/** How the core and the stream answer one write, in milliseconds after the write was sent. */
interface Timing {
  /** The core has saved the file. */
  readonly saveAfter: number
  /** The answer of the write arrives this long after the file was saved. */
  readonly answerAfter: number
  /** The event of the write arrives this long after the file was saved. */
  readonly eventAfter: number
  /** The first reads answer the file as it was before the last write. */
  readonly oldReads: number
  /** A read takes this long to answer. */
  readonly readTakes: number
  /** The core rejects the write with this reason and saves nothing. */
  readonly rejectedWith?: string
}

const NORMAL: Timing = { saveAfter: 40, answerAfter: 0, eventAfter: 100, oldReads: 0, readTakes: 30 }

const say = vi.fn()

const create = (timing: Timing) => {
  file.current = {
    app: {
      appearance: { theme: 'preset:numen', mode: 'system', interfaceScale: 1 },
      performance: { profile: 'balanced' },
    },
  }
  let before = clone(file.current)
  let oldReads = timing.oldReads
  const feed = createFeed()
  const core = {
    getSettings: vi.fn(async () => {
      const answer = oldReads > 0 ? before : file.current
      if (oldReads > 0) oldReads -= 1
      const written = JSON.stringify(answer)
      const app = declare()
      await new Promise((resolve) => setTimeout(resolve, timing.readTakes))
      return { written, path: '/config/numen.json', models: [], plugins: [], app }
    }),
    getSyncEnabled: vi.fn(async () => true),
    setSyncEnabled: vi.fn(async () => null),
    updateSettings: vi.fn(
      (edits: readonly SettingEdit[]) =>
        new Promise<void>((resolve, reject) => {
          setTimeout(() => {
            if (timing.rejectedWith) {
              reject(new Error(timing.rejectedWith))
              return
            }
            before = clone(file.current)
            for (const edit of edits) {
              const [owner, ...steps] = edit.at
              const last = steps.pop() as string
              const target = steps.reduce<Record<string, unknown>>((one, step) => {
                one[step] = { ...((one[step] as object | undefined) ?? {}) }
                return one[step] as Record<string, unknown>
              }, ((file.current[owner as string] ??= {}) as Record<string, unknown>))
              target[last] = JSON.parse(edit.value)
            }
            setTimeout(() => feed.push(edits), timing.eventAfter)
            setTimeout(resolve, timing.answerAfter)
          }, timing.saveAfter)
        }),
    ),
    watchSettings: vi.fn(() => feed.open()),
  } as unknown as SettingsPort
  const log = { getWriter: () => say } as unknown as MessageLog
  const settings = useSettings({
    core,
    words: WORDS,
    log,
    windowTabs: useWindowTabs(),
    onSizeChanged: vi.fn(),
  })
  const page = mount(SystemDetail, {
    props: { state: settings.configured.state },
    attachTo: document.body,
  })
  return { settings, page }
}

const getRow = (page: VueWrapper, title: string) => {
  const row = page.findAllComponents(SettingRow).find((one) => one.props('name') === title)
  if (!row) throw new Error(`no row is drawn for ${title}`)
  return row
}

const getShown = (page: VueWrapper, title: string, control: typeof Select | typeof Slider) =>
  getRow(page, title).findComponent(control).props('modelValue') as unknown

/** The values shown at every step of 10 ms for the time given, with the one shown right away first. */
const watchShown = async (read: () => unknown, ms: number): Promise<unknown[]> => {
  await nextTick()
  const shown = [read()]
  for (let step = 0; step < ms / 10; step++) {
    await vi.advanceTimersByTimeAsync(10)
    await nextTick()
    shown.push(read())
  }
  return shown
}

const getDistinct = (shown: readonly unknown[]): unknown[] =>
  shown.filter((one, index) => index === 0 || one !== shown[index - 1])

const start = async (timing: Timing) => {
  const made = create(timing)
  await made.settings.start()
  await vi.advanceTimersByTimeAsync(200)
  await nextTick()
  return made
}

const ORDERS: readonly [string, Timing][] = [
  ['the answer comes with the save and the event after it', NORMAL],
  ['the event comes before the answer', { ...NORMAL, eventAfter: 5, answerAfter: 60 }],
  ['the answer comes long after the event and the first read', { ...NORMAL, eventAfter: 0, answerAfter: 150 }],
  ['the first read after the answer gives the old value', { ...NORMAL, oldReads: 1 }],
  ['the reads are slow and the first gives the old value', { ...NORMAL, oldReads: 1, readTakes: 90 }],
]

beforeEach(() => {
  vi.useFakeTimers()
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe = vi.fn()
      unobserve = vi.fn()
      disconnect = vi.fn()
    },
  )
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
  document.head.querySelectorAll('style[data-appearance]').forEach((one) => one.remove())
})

describe('a list of the page System after the person chooses a value', () => {
  it.each(ORDERS)('shows only the new mode when %s', async (_, timing) => {
    const { settings, page } = await start(timing)
    expect(getShown(page, 'Light or dark', Select)).toBe('system')

    getRow(page, 'Light or dark').findComponent(Select).vm.$emit('update:modelValue', 'dark')
    const shown = await watchShown(() => getShown(page, 'Light or dark', Select), 600)

    expect(getDistinct(shown)).toStrictEqual(['dark'])
    settings.close()
    page.unmount()
  })

  it.each(ORDERS)('shows only the new performance profile when %s', async (_, timing) => {
    const { settings, page } = await start(timing)

    getRow(page, 'Performance profile').findComponent(Select).vm.$emit('update:modelValue', 'eco')
    const shown = await watchShown(() => getShown(page, 'Performance profile', Select), 600)

    expect(getDistinct(shown)).toStrictEqual(['eco'])
    settings.close()
    page.unmount()
  })

  it('shows only the theme of a plugin after the person chooses it', async () => {
    const { settings, page } = await start(NORMAL)

    getRow(page, 'Theme')
      .findComponent(Select)
      .vm.$emit('update:modelValue', 'plugin:md.example.themes/dusk')
    const shown = await watchShown(() => getShown(page, 'Theme', Select), 600)

    expect(getDistinct(shown)).toStrictEqual(['plugin:md.example.themes/dusk'])
    settings.close()
    page.unmount()
  })

  it('shows the old value once, after the reason is given, when the core rejects the write', async () => {
    say.mockClear()
    const { settings, page } = await start({ ...NORMAL, rejectedWith: 'the disk is full' })

    getRow(page, 'Light or dark').findComponent(Select).vm.$emit('update:modelValue', 'dark')
    const shown = await watchShown(() => getShown(page, 'Light or dark', Select), 600)

    expect(getDistinct(shown)).toStrictEqual(['dark', 'system'])
    expect(say).toHaveBeenCalledWith(expect.stringContaining('the disk is full'), 'error')
    settings.close()
    page.unmount()
  })
})

describe('a slider of the page System after the person moves it', () => {
  it.each(ORDERS)('shows only the new interface size when %s', async (_, timing) => {
    const { settings, page } = await start(timing)

    const slider = getRow(page, 'Interface size').findComponent(Slider)
    slider.vm.$emit('update:modelValue', 1.2)
    slider.vm.$emit('commit', 1.2)
    const shown = await watchShown(() => getShown(page, 'Interface size', Slider), 600)

    expect(getDistinct(shown)).toStrictEqual([1.2])
    settings.close()
    page.unmount()
  })

  it.each(ORDERS)('follows the last of two quick changes and never goes back when %s', async (_, timing) => {
    const { settings, page } = await start(timing)

    const slider = getRow(page, 'Interface size').findComponent(Slider)
    slider.vm.$emit('update:modelValue', 1.2)
    slider.vm.$emit('commit', 1.2)
    const first = await watchShown(() => getShown(page, 'Interface size', Slider), 20)
    slider.vm.$emit('update:modelValue', 1.4)
    slider.vm.$emit('commit', 1.4)
    const second = await watchShown(() => getShown(page, 'Interface size', Slider), 600)

    expect(getDistinct([...first, ...second])).toStrictEqual([1.2, 1.4])
    settings.close()
    page.unmount()
  })
})

describe('the window after the person chooses a theme or a size on the page System', () => {
  const getStyle = (is: string) =>
    document.head.querySelector(`style[data-appearance="${is}"]`)?.textContent ?? ''

  it('applies the theme of a plugin and keeps it after the event of the write', async () => {
    const { settings, page } = await start(NORMAL)

    getRow(page, 'Theme')
      .findComponent(Select)
      .vm.$emit('update:modelValue', 'plugin:md.example.themes/dusk')
    const applied = await watchShown(() => getStyle('theme'), 600)

    expect(getDistinct(applied).pop()).toContain('plugin:md.example.themes/dusk')
    expect(applied.slice(-20).every((one) => String(one).includes('dusk'))).toBe(true)
    settings.close()
    page.unmount()
  })

  it('applies the mode and keeps it after the event of the write', async () => {
    const { settings, page } = await start(NORMAL)

    getRow(page, 'Light or dark').findComponent(Select).vm.$emit('update:modelValue', 'dark')
    const applied = await watchShown(() => getStyle('mode'), 600)

    expect(getDistinct(applied).pop()).toContain('color-scheme: dark')
    expect(applied.slice(-20).every((one) => String(one).includes('dark'))).toBe(true)
    settings.close()
    page.unmount()
  })

  it('applies the interface size and keeps it after the event of the write', async () => {
    const { settings, page } = await start(NORMAL)

    const slider = getRow(page, 'Interface size').findComponent(Slider)
    slider.vm.$emit('update:modelValue', 1.4)
    slider.vm.$emit('commit', 1.4)
    const applied = await watchShown(() => getStyle('sizes'), 600)

    expect(getDistinct(applied).pop()).toContain('1.4')
    settings.close()
    page.unmount()
  })
})
