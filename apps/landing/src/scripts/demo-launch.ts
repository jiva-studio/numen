const loadDemo = () => import('../components/mountDemoWindow')

/** Loads the live demo into the hero when it is asked for, on the windows that show the launch button. */
export function initDemoLaunch(): void {
  const host = document.getElementById('hero-live-demo')
  const trigger = document.getElementById('demo-launch')
  let preloaded: ReturnType<typeof loadDemo> | null = null
  let isMounted = false

  // The stylesheet hides the button on small and touch windows, which is where the demo is not offered.
  const isOffered = (): boolean => trigger !== null && getComputedStyle(trigger).display !== 'none'

  const markFailed = (): void => {
    host?.removeAttribute('data-loading')
    host?.setAttribute('data-failed', 'true')
  }

  const preload = (): ReturnType<typeof loadDemo> | null => {
    if (!isOffered() || isMounted) return null
    preloaded ??= loadDemo()
    return preloaded
  }

  const launch = async (): Promise<void> => {
    if (!host || !isOffered() || isMounted) return
    isMounted = true
    host.setAttribute('data-loading', 'true')
    const mod = await (preload() ?? loadDemo())
    host.removeAttribute('data-loading')
    host.setAttribute('data-live', 'true')
    mod.mountDemoWindow('#hero-live-demo')
  }

  host?.addEventListener('pointerenter', () => {
    preload()?.catch(markFailed)
  })
  trigger?.addEventListener('focus', () => {
    preload()?.catch(markFailed)
  })
  trigger?.addEventListener('click', () => {
    launch().catch(markFailed)
  })
}
