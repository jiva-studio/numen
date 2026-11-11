import { describe, expect, it, vi } from 'vitest'
import { createCommandDeps, type CommandDepsOptions } from './createCommandDeps'

const createOptions = (override: Partial<CommandDepsOptions> = {}) => {
  const calls: unknown[][] = []
  const record =
    (name: string, answer?: unknown) =>
    (...args: unknown[]) => {
      calls.push([name, ...args])
      return answer
    }
  const options: CommandDepsOptions = {
    windowTabs: {
      tabs: {
        openTabOfKind: (...args: unknown[]) => {
          record('openTabOfKind')(...args)
          return Promise.resolve()
        },
      },
      panes: { requestClose: record('requestClose') },
    } as never,
    reached: {} as never,
    runs: {} as never,
    registry: { get: () => undefined },
    writeMessage: vi.fn(),
    settingPalette: { choose: () => Promise.resolve() },
    search: () => undefined,
    ...override,
  }
  return { deps: createCommandDeps(options), calls, options }
}

describe('createCommandDeps', () => {
  it('passes the row chosen for a setting to the adapter of the settings', async () => {
    const choose = vi.fn(async () => undefined)
    const { deps } = createOptions({ settingPalette: { choose } })

    await deps.settings.chooseSetting?.('setting:app:performance.profile', 'maximum')

    expect(choose).toHaveBeenCalledExactlyOnceWith('setting:app:performance.profile', 'maximum')
  })

  it('passes the choice of a setting of a plugin to the same adapter', async () => {
    const choose = vi.fn(async () => undefined)
    const { deps } = createOptions({ settingPalette: { choose } })

    await deps.settings.chooseSetting?.('setting:md.example.scanner:autoScan', 'on')

    expect(choose).toHaveBeenCalledExactlyOnceWith('setting:md.example.scanner:autoScan', 'on')
  })
})

describe('the commands reach the window', () => {
  it('routes where a command points and reports openTab errors', async () => {
    const writeMessage = vi.fn()
    const { deps, calls } = createOptions({ writeMessage })

    deps.navigator.openTab('settings')
    deps.navigator.closeTab('tab-1')
    expect(calls).toStrictEqual([
      ['openTabOfKind', 'settings'],
      ['requestClose', 'tab-1'],
    ])

    const failing = createOptions({
      writeMessage,
      windowTabs: {
        tabs: { openTabOfKind: () => Promise.reject(new Error('tab failed')) },
        panes: { requestClose: vi.fn() },
      } as never,
    })
    failing.deps.navigator.openTab('broken')
    await new Promise((r) => setTimeout(r, 0))
    expect(writeMessage).toHaveBeenCalledWith(expect.any(String), 'error')
  })

  it('searches with the search the window gave', () => {
    const search = vi.fn()

    createOptions({ search }).deps.navigator.search()

    expect(search).toHaveBeenCalledOnce()
  })

  it('keeps the runs it was given', () => {
    const runs = { name: 'runs' } as never

    expect(createOptions({ runs }).deps.runSupport).toBe(runs)
  })
})
