import { afterEach, describe, expect, it } from 'vitest'
import { CATALOG_STATE } from '@numen/core-client'
import {
  createLostConnection,
  createMockCatalogEntry,
  installCoreAnswers,
} from '@numen/core-client/testing'
import { listCatalog } from './catalog'

let restore: (() => void) | undefined

afterEach(() => {
  restore?.()
  restore = undefined
})

describe('the entries of the catalog', () => {
  it('arrive with the category, the feature and the kinds of tab of each plugin', async () => {
    restore = installCoreAnswers({
      pluginCatalog: {
        listCatalog: () => ({
          entries: [
            createMockCatalogEntry('notes', {
              category: 'Editors',
              feature: 'Markdown notes',
              tabKinds: ['note', { kind: 'reader:markdown', title: 'Markdown' }],
            }),
          ],
        }),
      },
    })

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
        ],
        sourceFailures: [],
      },
    })
  })

  it('mark only an available entry as offered', async () => {
    restore = installCoreAnswers({
      pluginCatalog: {
        listCatalog: () => ({
          entries: [
            createMockCatalogEntry('a', { state: CATALOG_STATE.AVAILABLE }),
            createMockCatalogEntry('b', { state: CATALOG_STATE.DISABLED }),
          ],
        }),
      },
    })

    const answer = await listCatalog()

    expect(answer.ok && answer.value.entries.map((one) => one.isAvailable)).toStrictEqual([
      true,
      false,
    ])
  })

  it('name the sources that could not be read', async () => {
    restore = installCoreAnswers({
      pluginCatalog: {
        listCatalog: () => ({
          entries: [],
          sourceErrors: [{ source: 'sample-index', message: 'timed out' }],
        }),
      },
    })

    const answer = await listCatalog()

    expect(answer.ok && answer.value.sourceFailures).toStrictEqual([
      { source: 'sample-index', message: 'timed out' },
    ])
  })

  it('answer why the catalog cannot be read', async () => {
    restore = installCoreAnswers({
      pluginCatalog: {
        listCatalog: () => {
          throw createLostConnection('gone')
        },
      },
    })

    const answer = await listCatalog()

    expect(answer).toStrictEqual({ ok: false, error: 'Numen is not responding; retrying' })
  })
})
