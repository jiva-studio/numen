/**
 * Bootstraps and disposes window background processes and streams.
 */
import { onMounted, onUnmounted, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { waitFor } from './waitFor'

export interface BootstrapDeps {
  loadVaults(): Promise<void>
  startSettings(): Promise<void>
  startWindow(): Promise<void>
  startEditing(): void
  startLayout(): Promise<void>
  startPlugins?(): Promise<void>
  closeWindow(): void
  closeEditing(): void
  closeTabs(): void
  closeSettings(): void
  closePlugins?(): void | Promise<void>
  /** Receives the failure of a start or a close that is not awaited. */
  reportError(error: unknown): void
}

/** How long the window waits for the plugins before it opens the tabs without the ones still starting. */
export const PLUGINS_WAIT_MS = 2_000

/** How long one of the steps the workspace depends on may take before the window says it could not load. */
export const STEP_LIMIT_MS = 10_000

async function startPluginsWithinLimit(deps: BootstrapDeps): Promise<void> {
  const starting = deps.startPlugins?.()
  if (!starting) return
  runInBackground(starting, deps.reportError)
  await waitFor(starting, PLUGINS_WAIT_MS)
}

async function withinLimit<T>(what: string, work: Promise<T>, limitMs: number): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined
  const late = new Promise<never>((_, reject) => {
    timer = setTimeout(
      () => reject(new Error(`${what} did not answer in ${limitMs / 1000} s`)),
      limitMs,
    )
  })
  /* Stryker disable BlockStatement,CallExpression: timer cleanup after Promise.race settles */
  try {
    return await Promise.race([work, late])
  } finally {
    clearTimeout(timer)
  }
  /* Stryker restore BlockStatement,CallExpression */
}

export function useAppBootstrap(deps: BootstrapDeps, stepLimitMs: number = STEP_LIMIT_MS) {
  /** Why the workspace did not load, shown with a retry; empty while it loads and once it has. */
  const failure = ref('')
  let attempt = 0

  const run = async (): Promise<void> => {
    // Stryker disable next-line UpdateOperator: generation counter only needs a distinct value per run
    const mine = ++attempt
    failure.value = ''
    try {
      await withinLimit('The vault list', deps.loadVaults(), stepLimitMs)
      await withinLimit('The settings request', deps.startSettings(), stepLimitMs)
      await startPluginsWithinLimit(deps)
      await withinLimit('The tab restore', deps.startLayout(), stepLimitMs)
    } catch (error) {
      if (mine === attempt) failure.value = error instanceof Error ? error.message : String(error)
      return
    }
    if (mine !== attempt) return
    runInBackground(deps.startWindow(), deps.reportError)
    deps.startEditing()
  }

  onMounted(run)

  onUnmounted(() => {
    // Stryker disable next-line UpdateOperator: generation counter only needs a distinct value per run
    attempt++
    runInBackground(Promise.resolve(deps.closePlugins?.()), deps.reportError)
    deps.closeWindow()
    deps.closeEditing()
    deps.closeTabs()
    deps.closeSettings()
  })

  return { failure, retry: run }
}
