import type { AnyTabKind } from '@numen/plugin-loader'

/** The kind the window draws: the main-pane kind of the lowest order, or null where there is none. */
export function findFrontKind(kinds: readonly AnyTabKind[]): AnyTabKind | null {
  const mainKinds = kinds.filter((one) => one.display?.placement?.pane === 'main')
  mainKinds.sort(
    // Stryker disable next-line OptionalChaining: mainKinds only holds kinds with defined placement
    (a, b) => (a.display?.placement?.order ?? 0) - (b.display?.placement?.order ?? 0),
  )
  return mainKinds[0] ?? null
}
