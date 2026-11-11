/**
 * The keys of a window: the keybindings of the application and of the plugins, as they are matched
 * against a key event. An owner registers its keybindings and unregisters them with itself.
 *
 * Among the keybindings of one key whose condition holds, the winner is the one whose condition reads
 * the most variables (a condition of none, or an empty one, reads none), then one of the application
 * over one of a plugin, then the one registered first. Another keybinding whose condition reads as many
 * variables as the winner's, and that runs another command, is a conflict, reported once.
 */
import type { WhenCondition, WhenValues, WhenReader } from '../when'
import { APP_COMMAND_OWNER, formatCommandId } from '../commandId'
import { isKeyEvent, parseKey, type ParsedKey } from './parseKey'

/** A keybinding as a manifest or the application declares it. */
export interface KeybindingSource {
  readonly key: string
  readonly command: string
  readonly values?: Readonly<Record<string, string>> | undefined
  readonly when?: string | undefined
}

/** A keybinding as the registry holds it: the key read once, and the id of the command in full. */
export interface Keybinding {
  readonly owner: string
  readonly key: string
  readonly parsed: ParsedKey
  readonly command: string
  readonly values: Readonly<Record<string, string>>
  readonly when: string
  readonly condition: WhenCondition
}

export interface KeybindingRegistryDeps {
  readonly isMac: () => boolean
  readonly whenReader: WhenReader
  readonly report?: ((owner: string, message: string) => void) | undefined
}

export interface KeybindingRegistry {
  /** Registers the keybindings of the owner, in place of those it registered before. */
  register(owner: string, sources: readonly KeybindingSource[]): void
  unregister(owner: string): void
  /**
   * The keybinding the event runs, or undefined. Only a keybinding whose condition holds over the
   * context, and that `canRun` accepts if given, is a candidate. It changes nothing: the caller decides
   * whether to prevent the default of the event.
   */
  match(
    event: KeyboardEvent,
    context: WhenValues,
    canRun?: (keybinding: Keybinding) => boolean,
  ): Keybinding | undefined
  readonly getKeyFor: (
    command: string,
    values?: Readonly<Record<string, string>>,
  ) => string | undefined
}

const getValuesKey = (values: Readonly<Record<string, string>>): string =>
  JSON.stringify(Object.entries(values).sort(([a], [b]) => a.localeCompare(b)))

const isSameRun = (a: Keybinding, b: Keybinding): boolean =>
  a.command === b.command && getValuesKey(a.values) === getValuesKey(b.values)

const getSpecificity = (one: Keybinding): number => one.condition.names.length

function isStronger(a: Keybinding, b: Keybinding): boolean {
  const byCondition = getSpecificity(a) - getSpecificity(b)
  if (byCondition !== 0) return byCondition > 0
  return a.owner === APP_COMMAND_OWNER && b.owner !== APP_COMMAND_OWNER
}

type Report = (owner: string, message: string) => void

function createReport(sink: Report | undefined): Report {
  const reported = new Set<string>()
  return (owner, message) => {
    const key = `${owner}\n${message}`
    if (reported.has(key)) return
    reported.add(key)
    sink?.(owner, message)
  }
}

function readKeybinding(
  owner: string,
  source: KeybindingSource,
  env: { readonly whenReader: WhenReader; readonly report: Report },
): Keybinding | undefined {
  const parsed = parseKey(source.key)
  if (!parsed) {
    env.report(owner, `keybindings: the key "${source.key}" cannot be read`)
    return undefined
  }
  const when = source.when ?? ''
  const condition = env.whenReader.read(when)
  if (!condition.ok) {
    env.report(
      owner,
      `keybindings: the condition "${when}" of "${source.key}" cannot be read: ${condition.error}`,
    )
    return undefined
  }
  return {
    owner,
    key: source.key,
    parsed,
    command: source.command.includes(':') ? source.command : formatCommandId(owner, source.command),
    values: source.values ?? {},
    when,
    condition: condition.value,
  }
}

function reportConflict(winner: Keybinding, others: readonly Keybinding[], report: Report): void {
  const rivals = others.filter(
    (one) => getSpecificity(one) === getSpecificity(winner) && !isSameRun(one, winner),
  )
  if (rivals.length === 0) return
  const names = rivals.map((one) => `${one.command} (${one.owner})`).join(', ')
  report(
    winner.owner,
    `keybindings: the key "${winner.key}" is bound to ${winner.command} and to ${names} under conditions of the same width; ${winner.command} wins`,
  )
}

export function createKeybindingRegistry(deps: KeybindingRegistryDeps): KeybindingRegistry {
  const { isMac, whenReader } = deps
  let keybindings: readonly Keybinding[] = []
  const report = createReport(deps.report)

  return {
    register: (owner, sources) => {
      const added = sources.flatMap((source) => readKeybinding(owner, source, { whenReader, report }) ?? [])
      keybindings = [...keybindings.filter((one) => one.owner !== owner), ...added]
    },
    unregister: (owner) => {
      keybindings = keybindings.filter((one) => one.owner !== owner)
    },
    match: (event, context, canRun) => {
      const isMacNow = isMac()
      const candidates = keybindings.filter(
        (one) =>
          isKeyEvent(one.parsed, event, isMacNow) &&
          one.condition.evaluate(context) &&
          (canRun?.(one) ?? true),
      )
      const winner = candidates.reduce<Keybinding | undefined>(
        (best, one) => (best === undefined || isStronger(one, best) ? one : best),
        undefined,
      )
      if (winner) reportConflict(winner, candidates, report)
      return winner
    },
    getKeyFor: (command, values = {}) => {
      const key = getValuesKey(values)
      return keybindings.find((one) => one.command === command && getValuesKey(one.values) === key)
        ?.key
    },
  }
}
