/**
 * The plugin catalog as the core answers it to the placeholder of a tab kind.
 */
import type { CatalogEntry } from '@numen/core-client'
import type { CoreAnswers } from '@numen/core-client/testing'
import { createTabKindPluginFinder } from '@/entities/tab'
import { createAnsweredClients } from './coreAnswers'

/**
 * Answers the question which plugin provides a kind of tab from `entries`, and records the kinds
 * it is asked about, in order, in `lookups`. A kind no entry declares is answered as not found.
 * `before` runs before each answer, and the answer waits for what it returns. `answers` are the
 * answers of the core, and `findTabKindPlugin` asks them through the real client.
 */
export const answerTabKinds = (
  entries: readonly CatalogEntry[],
  before: () => Promise<void> = () => Promise.resolve(),
) => {
  const lookups: string[] = []
  const answers: CoreAnswers = {
    pluginCatalog: {
      findCatalogEntryByTabKind: async (request) => {
        lookups.push(request.kind)
        await before()
        const entry = entries.find((one) =>
          one.contributes?.tabKinds.some((tabKind) => tabKind.kind === request.kind),
        )
        return entry ? { entry } : {}
      },
    },
  }
  const findTabKindPlugin = createTabKindPluginFinder(createAnsweredClients(answers).pluginCatalog)
  return { answers, lookups, findTabKindPlugin }
}
