import { describe, expect, it, vi } from 'vitest'
import { reportFailure, runInBackground } from '@numen/ui'
import { watchCloudVaults } from './watchCloudVaults'
import { useCloudVaults } from './useCloudVaults'

const CLOUD_REFRESH_INTERVAL_MS = 5 * 60 * 1000
type CloudWatchPorts = NonNullable<Parameters<typeof watchCloudVaults>[1]>

function setup(visible = true) {
  const state = {
    visible,
    wake: undefined as (() => void) | undefined,
    tick: undefined as (() => void) | undefined,
  }
  const stopped = { wake: 0, timer: 0 }
  const ports: CloudWatchPorts = {
    isVisible: () => state.visible,
    onWake: (listener) => {
      state.wake = listener
      return () => {
        stopped.wake += 1
      }
    },
    every: (task, ms) => {
      expect(ms).toBe(CLOUD_REFRESH_INTERVAL_MS)
      state.tick = task
      return () => {
        state.tick = undefined
        stopped.timer += 1
      }
    },
  }
  const refresh = vi.fn()
  return { state, stopped, ports, refresh }
}

describe('watching the cloud vaults', () => {
  it('refreshes at every tick of the interval while the window is visible', () => {
    const { state, ports, refresh } = setup()
    watchCloudVaults(refresh, ports)

    state.tick?.()
    state.tick?.()

    expect(refresh).toHaveBeenCalledTimes(2)
  })

  it('refreshes when the window gets focus', () => {
    const { state, ports, refresh } = setup()
    watchCloudVaults(refresh, ports)

    state.wake?.()

    expect(refresh).toHaveBeenCalledTimes(1)
  })

  it('stops the interval when the window is hidden and starts it again when it is shown', () => {
    const { state, ports, refresh } = setup()
    watchCloudVaults(refresh, ports)

    state.visible = false
    state.wake?.()
    expect(state.tick).toBeUndefined()
    expect(refresh).not.toHaveBeenCalled()

    state.visible = true
    state.wake?.()
    expect(state.tick).toBeDefined()
    expect(refresh).toHaveBeenCalledTimes(1)
  })

  it('starts no interval while the window is hidden', () => {
    const { state, ports, refresh } = setup(false)
    watchCloudVaults(refresh, ports)

    expect(state.tick).toBeUndefined()
  })

  it('stops the listening and the interval', () => {
    const { stopped, ports, refresh } = setup()

    watchCloudVaults(refresh, ports)()

    expect(stopped).toStrictEqual({ wake: 1, timer: 1 })
  })

  it('does not overlap two refreshes of the list', async () => {
    const { state, ports } = setup()
    let release: (() => void) | undefined
    const list = vi
      .fn<() => Promise<readonly []>>()
      .mockImplementationOnce(
        () =>
          new Promise<readonly []>((resolve) => {
            release = () => resolve([])
          }),
      )
      .mockResolvedValue([])
    const cloud = useCloudVaults({ list })
    watchCloudVaults(() => runInBackground(cloud.refresh(), reportFailure), ports)

    state.wake?.()
    state.tick?.()
    state.wake?.()
    release?.()
    await vi.waitFor(() => expect(list).toHaveBeenCalledTimes(1))
    await cloud.refresh()

    expect(list).toHaveBeenCalledTimes(2)
  })
})
