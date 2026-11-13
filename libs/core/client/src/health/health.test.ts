/** The health items the core keeps, in the windows' terms, and a report made for a frontend. */
import { afterEach, describe, expect, it } from 'vitest'
import { HealthSeverity, type ReportHealthRequest } from '@numen/protocol'
import { pluginHealth } from '../clients'
import { installCoreAnswers } from '../testing'
import { createHealthClient } from './health'

const health = createHealthClient(pluginHealth)

let restore: (() => void) | undefined

afterEach(() => {
  restore?.()
  restore = undefined
})

const readFirst = async <T>(stream: AsyncIterable<T>): Promise<T | undefined> => {
  for await (const one of stream) return one
  return undefined
}

describe('the health items', () => {
  it('arrive with their severity and their action', async () => {
    restore = installCoreAnswers({
      pluginHealth: {
        watchHealth: async function* () {
          yield {
            items: [
              {
                pluginId: 'org.example.atlas',
                key: 'connection',
                severity: HealthSeverity.ERROR,
                text: 'Choose a connection',
                detail: 'The atlas has no connection to ask.',
                action: {
                  command: 'app:settings.open',
                  values: { plugin: 'org.example.atlas', setting: 'source' },
                  title: '',
                },
              },
              {
                pluginId: 'org.example.atlas',
                key: 'quota',
                severity: HealthSeverity.WARNING,
                text: 'Half of the quota is used',
                detail: '',
              },
            ],
          }
        },
      },
    })

    expect(await readFirst(health.watchHealth(new AbortController().signal))).toStrictEqual([
      {
        pluginId: 'org.example.atlas',
        key: 'connection',
        severity: 'error',
        text: 'Choose a connection',
        detail: 'The atlas has no connection to ask.',
        action: {
          command: 'app:settings.open',
          values: { plugin: 'org.example.atlas', setting: 'source' },
          title: '',
        },
      },
      {
        pluginId: 'org.example.atlas',
        key: 'quota',
        severity: 'warning',
        text: 'Half of the quota is used',
        detail: '',
        action: undefined,
      },
    ])
  })

  it('are reported under the plugin the window names', async () => {
    const reported: ReportHealthRequest[] = []
    restore = installCoreAnswers({
      pluginHealth: {
        reportHealth: (request) => {
          reported.push(request)
          return {}
        },
      },
    })

    await health.reportHealth('org.example.atlas', {
      key: 'connection',
      severity: 'error',
      text: 'Choose a connection',
      action: { command: 'app:settings.open', values: { setting: 'source' } },
    })

    expect(reported.map((one) => one.item)).toMatchObject([
      {
        pluginId: 'org.example.atlas',
        key: 'connection',
        severity: HealthSeverity.ERROR,
        text: 'Choose a connection',
        detail: '',
        action: { command: 'app:settings.open', values: { setting: 'source' }, title: '' },
      },
    ])
  })
})
