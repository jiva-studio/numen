/**
 * One link as its tab shows it: the address, the text fetched for it and the copy kept of its
 * video, read again whenever the work on it ends. A link whose text was never fetched is fetched
 * once the tab opens.
 */
import { computed, ref, shallowRef } from 'vue'
import { splitParagraphs } from '../lib/text'
import { links as servedLinks } from '../api/wire'
import type { LinkFacts, Links, LinkSource, Outcome } from '../types'

export function useLink(links: Links | undefined, path: string) {
  const activeLinks = links ?? servedLinks
  const url = ref('')
  const text = ref('')
  const source = shallowRef<LinkSource | null>(null)
  const embedSource = ref('')
  const isCopyable = ref(false)
  const isTextDeleted = ref(false)
  const isWorking = ref(false)
  const isLoaded = ref(false)
  const error = ref('')
  let turn = 0
  let isFetchAsked = false

  const load = async (): Promise<void> => {
    const mine = ++turn
    try {
      const facts = await activeLinks.getLink(path)
      if (mine !== turn) return
      if (!facts.ok) {
        error.value = facts.error
        return
      }
      const [article, [played, framed]] = await Promise.all([
        activeLinks.readArticle(path),
        readPlayback(activeLinks, path, facts.value),
      ])
      if (mine !== turn) return
      url.value = facts.value.url
      isCopyable.value = facts.value.isCopyable
      isTextDeleted.value = facts.value.isTextDeleted
      text.value = article
      source.value = played
      embedSource.value = framed
      error.value = ''
      if (!article && !facts.value.isTextDeleted && !isWorking.value && !isFetchAsked) {
        isFetchAsked = true
        await run(() => activeLinks.fetchText(path, false))
      }
    } catch (failure) {
      if (mine === turn) error.value = describeFailure(failure)
    } finally {
      if (mine === turn) isLoaded.value = true
    }
  }

  const run = async (start: () => Promise<Outcome<null>>): Promise<void> => {
    try {
      const outcome = await start()
      if (!outcome.ok) error.value = outcome.error
    } catch (failure) {
      error.value = describeFailure(failure)
    }
  }

  const stop = activeLinks.watchWork(path, (working) => {
    isWorking.value = working
    if (!working) void load()
  })
  void load()

  return {
    url,
    paragraphs: computed(() => splitParagraphs(text.value)),
    source,
    /** Where the page that frames the video's player is loaded from, while no copy is kept. */
    embedSource,
    status: { isCopyable, isTextDeleted, isWorking, isLoaded },
    error,
    actions: {
      fetchText: () => run(() => activeLinks.fetchText(path, true)),
      copyVideo: () => run(() => activeLinks.copyVideo(path)),
      deleteCopy: async () => {
        await run(() => activeLinks.deleteCopy(path))
        await load()
      },
    },
    close: stop,
  }
}

/** Where the copy is played from, or else where the page that frames the player is loaded from. */
const readPlayback = async (
  links: Links,
  path: string,
  facts: LinkFacts,
): Promise<readonly [LinkSource | null, string]> => {
  if (facts.copySize > 0) return [await links.getSource(path), '']
  if (facts.embedPage) return [null, await links.getPageSource(facts.embedPage)]
  return [null, '']
}

const describeFailure = (failure: unknown): string =>
  failure instanceof Error ? failure.message : String(failure)

export type LinkState = ReturnType<typeof useLink>
