/**
 * Bootstraps and disposes window background processes and streams.
 */
import { onMounted, onUnmounted, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { INTERFACE_WAIT_MS } from '@numen/plugin-loader'
import { waitFor } from './waitFor'
import type { Caught } from '@/shared/api/errors'

export interface BootstrapDeps {
  loadVaults(): Promise<void>
  startSettings(): Promise<void>
  startWindow(): Promise<void>
  startEditing(): void
  startLayout(): Promise<void>
  /** Starts every plugin; settles once all have started or failed. */
  startPlugins?(): Promise<void>
  /** Settles once the plugins of the window slots have started or failed. */
  waitForInterfacePlugins?(): Promise<void>
  closeWindow(): void
  closeEditing(): void
  closeTabs(): void
  closeSettings(): void
  closePlugins?(): void | Promise<void>
  /** Receives the failure of a start or a close that is not awaited. */
  reportError(this: void, error: Caught): void
}

/** How long one of the steps the workspace depends on may take before the window says it could not load. */
const STEP_LIMIT_MS = 10_000

async function startPluginsWithinLimit(deps: BootstrapDeps): Promise<void> {
  const starting = deps.startPlugins?.()
  if (!starting) return
  runInBackground(starting, deps.reportError)
  const interfaceStarted = deps.waitForInterfacePlugins?.() ?? starting
  await waitFor(interfaceStarted, INTERFACE_WAIT_MS)
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

export function createAppBootstrap(deps: BootstrapDeps, stepLimitMs: number = STEP_LIMIT_MS) {
  /**
   * True once the plugins of the window slots have started, INTERFACE_WAIT_MS has passed since the window
   * mounted, or the start failed: the window is drawn from then on.
   */
  const isStarted = ref(false)
  /** Why the workspace did not load, shown with a retry; empty while it loads and once it has. */
  const failure = ref('')
  let attempt = 0
  let drawTimer: ReturnType<typeof setTimeout> | undefined

  const run = async (): Promise<void> => {
    // Stryker disable next-line UpdateOperator: generation counter only needs a distinct value per run
    const mine = ++attempt
    failure.value = ''
    try {
      await withinLimit('The vault list', deps.loadVaults(), stepLimitMs)
      await withinLimit('The settings request', deps.startSettings(), stepLimitMs)
      await startPluginsWithinLimit(deps)
      isStarted.value = true
      await withinLimit('The tab restore', deps.startLayout(), stepLimitMs)
    } catch (error) {
      isStarted.value = true
      if (mine === attempt) failure.value = error instanceof Error ? error.message : String(error)
      return
    }
    if (mine !== attempt) return
    runInBackground(deps.startWindow(), deps.reportError)
    deps.startEditing()
  }

  onMounted(() => {
    drawTimer = setTimeout(() => (isStarted.value = true), INTERFACE_WAIT_MS)
    return run()
  })

  onUnmounted(() => {
    clearTimeout(drawTimer)
    // Stryker disable next-line UpdateOperator: generation counter only needs a distinct value per run
    attempt++
    runInBackground(Promise.resolve(deps.closePlugins?.()), deps.reportError)
    deps.closeWindow()
    deps.closeEditing()
    deps.closeTabs()
    deps.closeSettings()
  })

  return { failure, isStarted, retry: run }
}
