/**
 * Every command of the application that a plugin names in a slot entry or a keybinding is known to
 * the palette, has a handler, and starts through the runner where it comes with values.
 */
import { describe, expect, it, vi } from 'vitest'
import { isSharedCommand } from '@numen/plugin-loader'
import { createInvocation, runInvocation } from '@/features/command-palette'
import { createAppCommands } from '@/testing/appCommands'
import { createRealWindow, NOTE, readAllManifests } from '@/testing/realCommands'

type Values = Readonly<Record<string, string>>

interface Named {
  readonly command: string
  readonly values: Values
}

const MANIFESTS = readAllManifests()

const listNamedCommands = (): readonly Named[] => {
  const found = new Map<string, Named>()
  for (const manifest of MANIFESTS) {
    const { slots = [], keybindings = [] } = manifest.contributes
    for (const one of [...slots, ...keybindings]) {
      const command = one.command
      if (!command?.startsWith('app:') || isSharedCommand(command)) continue
      const values = one.values ?? {}
      found.set(`${command}${JSON.stringify(values)}`, { command, values })
    }
  }
  return [...found.values()]
}

const NAMED = listNamedCommands()
const TABLE = createAppCommands()

const setup = () => {
  const events: string[] = []
  const navigateTab = vi.fn(async () => undefined)
  const emitEvent = (event: string) => events.push(event)
  return { events, ...createRealWindow({ manifests: MANIFESTS, navigateTab, emitEvent }) }
}

describe('the commands of the application the manifests name', () => {
  it('are named by at least the reveal of a note', () => {
    expect(NAMED.map((one) => one.command)).toContain('app:file.reveal')
  })

  it.each(NAMED.map((one) => [one.command, one] as const))(
    '%s is known to the palette',
    (_, one) => {
      expect(TABLE.some((row) => row.id === one.command)).toBe(true)
    },
  )

  it.each(NAMED.map((one) => [one.command, one] as const))('%s has a handler', async (_, one) => {
    const window = setup()
    const invocation = createInvocation(one.command, NOTE)

    await runInvocation(invocation, window.used.commandDeps)

    expect(window.events).toStrictEqual([`onCommand:${one.command}`])
  })

  it('start through the runner with their values, over a note', async () => {
    const withValues = NAMED.filter(
      (one) =>
        Object.keys(one.values).length > 0 && !TABLE.find((row) => row.id === one.command)?.input,
    )
    expect(withValues).not.toStrictEqual([])

    for (const one of withValues) {
      const window = setup()
      await window.used.pluginRuns.runCommandToEnd(one.command, NOTE, one.values)
      expect(window.events, one.command).toStrictEqual([`onCommand:${one.command}`])
    }
  })
})
