import { describe, expect, it, vi } from 'vitest'
import { CATALOG_STATE } from '@numen/core-client'
import { createLostConnection, createMockCatalogEntry } from '@numen/core-client/testing'
import { createAnsweredApi, createAnsweredClients } from '@/testing/coreAnswers'
import { createCatalogReader } from './catalog'
import { createHealthApi } from './health'
import { createLogHost } from './logs'
import { createProcessHost } from './processes'

describe('the entries of the catalog', () => {
  it('arrive with the category, the feature and the kinds of tab of each plugin', async () => {
    const { listCatalog } = createAnsweredApi({
      pluginCatalog: {
        listCatalog: () => ({
          entries: [
            createMockCatalogEntry('notes', {
              category: 'Editors',
              feature: 'Markdown notes',
              tabKinds: ['note', { kind: 'reader:markdown', title: 'Markdown' }],
            }),
            {
              id: 'legacy',
              name: 'legacy',
              version: '1.0.0',
              description: '',
              category: 'Tools',
              contributes: { category: '', feature: '', tabKinds: [] },
              state: CATALOG_STATE.AVAILABLE,
            },
            {
              id: 'bare',
              name: 'bare',
              version: '1.0.0',
              description: '',
              category: '',
              state: CATALOG_STATE.DISABLED,
            },
          ],
        }),
      },
    }).plugins

    const answer = await listCatalog()

    expect(answer).toStrictEqual({
      ok: true,
      value: {
        entries: [
          {
            id: 'notes',
            name: 'notes',
            version: '1.0.0',
            description: '',
            category: 'Editors',
            feature: 'Markdown notes',
            tabKinds: ['note', 'reader:markdown'],
            isAvailable: false,
          },
          {
            id: 'legacy',
            name: 'legacy',
            version: '1.0.0',
            description: '',
            category: 'Tools',
            feature: '',
            tabKinds: [],
            isAvailable: true,
          },
          {
            id: 'bare',
            name: 'bare',
            version: '1.0.0',
            description: '',
            category: '',
            feature: '',
            tabKinds: [],
            isAvailable: false,
          },
        ],
        sourceFailures: [],
      },
    })
  })

  it('mark only an available entry as offered', async () => {
    const { listCatalog } = createAnsweredApi({
      pluginCatalog: {
        listCatalog: () => ({
          entries: [
            createMockCatalogEntry('a', { state: CATALOG_STATE.AVAILABLE }),
            createMockCatalogEntry('b', { state: CATALOG_STATE.DISABLED }),
          ],
        }),
      },
    }).plugins

    const answer = await listCatalog()

    expect(answer.ok && answer.value.entries.map((one) => one.isAvailable)).toStrictEqual([
      true,
      false,
    ])
  })

  it('name the sources that could not be read', async () => {
    const { listCatalog } = createAnsweredApi({
      pluginCatalog: {
        listCatalog: () => ({
          entries: [],
          sourceErrors: [{ source: 'sample-index', message: 'timed out' }],
        }),
      },
    }).plugins

    const answer = await listCatalog()

    expect(answer.ok && answer.value.sourceFailures).toStrictEqual([
      { source: 'sample-index', message: 'timed out' },
    ])
  })

  it('answer why the catalog cannot be read, or when it lists nothing', async () => {
    const { listCatalog } = createAnsweredApi({
      pluginCatalog: {
        listCatalog: () => {
          throw createLostConnection('gone')
        },
      },
    }).plugins

    expect(await listCatalog()).toStrictEqual({
      ok: false,
      error: 'Numen is not responding; retrying',
    })

    const readEmpty = createCatalogReader({
      list: () =>
        Promise.resolve({
          ok: false,
          error: { error: 'notFound', message: 'not found', sourceErrors: [] },
        }),
    } as never)
    expect(await readEmpty()).toStrictEqual({ ok: false, error: 'it lists nothing' })
  })
})

describe('createHealthApi and createProcessHost', () => {
  it('reports and clears plugin health items', async () => {
    const reported: object[] = []
    const cleared: object[] = []
    const health = createHealthApi(
      createAnsweredClients({
        pluginHealth: {
          reportHealth: (req) => {
            reported.push(req)
            return {}
          },
          clearHealth: (req) => {
            cleared.push(req)
            return {}
          },
        },
      }).pluginHealth,
    )

    await health.healthReports.reportHealth('org.example.atlas', {
      key: 'quota',
      severity: 'warning',
      text: 'Half used',
    })
    await health.healthReports.clearHealth('org.example.atlas', 'quota')

    expect(reported).toMatchObject([
      { item: { pluginId: 'org.example.atlas', key: 'quota', text: 'Half used' } },
    ])
    expect(cleared).toMatchObject([{ pluginId: 'org.example.atlas', key: 'quota' }])
  })

  it('watches running plugin processes until disposed', async () => {
    const host = createProcessHost(
      createAnsweredClients({
        pluginProcessService: {
          watchPluginProcesses: async function* () {
            yield { processes: [], timeUnixMs: 42n }
          },
        },
      }).pluginProcessService,
    )
    const seen: number[] = []
    const subscription = host.watch((_processes, time) => {
      seen.push(time)
      subscription.dispose()
    })

    await vi.waitFor(() => expect(seen).toStrictEqual([42]))
  })

  it('watches and writes structured logs for a plugin', async () => {
    const written: object[] = []
    const getLogs = createLogHost(
      createAnsweredClients({
        logService: {
          watchLogs: async function* () {
            yield {
              entries: [
                {
                  timeUnixMs: 100n,
                  level: 2,
                  source: 'core',
                  message: 'started',
                  propertiesJson: { ok: 'true' },
                },
              ],
            }
          },
          writeLog: (req) => {
            written.push(req)
            return {}
          },
        },
      }).logService,
    )
    const host = getLogs('md.numen.tools.logs')
    const seen: string[] = []
    const subscription = host.watch((entries) => {
      seen.push(...entries.map((one) => one.message))
      subscription.dispose()
    })

    await vi.waitFor(() => expect(seen).toStrictEqual(['started']))
    await host.write({ level: 'debug', message: 'opened tab', properties: { tab: 'logs' } })
    expect(written).toMatchObject([
      {
        level: 1,
        source: 'md.numen.tools.logs',
        message: 'opened tab',
        propertiesJson: { tab: '"logs"' },
      },
    ])
  })
})
