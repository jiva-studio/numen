/**
 * The plugin catalog as the core answers it to the placeholder of a tab kind.
 */
import type { CatalogEntry } from '@numen/core-client'
import { answerCore } from './coreAnswers'

/**
 * Answers the question which plugin provides a kind of tab from `entries` until the test ends,
 * and returns the kinds it is asked about, in order. A kind no entry declares is answered as not found.
 * `before` runs before each answer, and the answer waits for what it returns.
 */
export const answerTabKinds = (
  entries: readonly CatalogEntry[],
  before: () => Promise<void> = () => Promise.resolve(),
): readonly string[] => {
  const lookups: string[] = []
  answerCore({
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
  })
  return lookups
}
