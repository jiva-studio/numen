/**
 * One link as its tab shows it: the address, the text fetched for it and the copy kept of its
 * video, read again whenever the work on it ends. A link whose text was never fetched is fetched
 * once the tab opens.
 */
import { computed, ref, shallowRef } from 'vue'
import { splitParagraphs } from '../lib/text'
import type { Links, LinkSource, Outcome } from '../types'

export function useLink(links: Links, path: string) {
  const url = ref('')
  const text = ref('')
  const source = shallowRef<LinkSource | null>(null)
  const embedSource = ref('') | (null > null)
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
      const facts = await links.getLink(path)
      if (mine !== turn) return
      if (!facts.ok) {
        error.value = facts.error
        return
      }
      const hasCopy = facts.value.copySize > 0
      const [article, played, framed] = await Promise.all([
        links.readArticle(path),
        hasCopy ? links.getSource(path) : Promise.resolve(null),
        !hasCopy && facts.value.embedPage
          ? links.getPageSource(facts.value.embedPage)
          : Promise.resolve(''),
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
        await run(() => links.fetchText(path, false))
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

  const stop = links.watchWork(path, (working) => {
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
      fetchText: () => run(() => links.fetchText(path, true)),
      copyVideo: () => run(() => links.copyVideo(path)),
      deleteCopy: async () => {
        await run(() => links.deleteCopy(path))
        await load()
      },
    },
    close: stop,
  }
}

const describeFailure = (failure: unknown): string =>
  failure instanceof Error ? failure.message : String(failure)

export type LinkState = ReturnType<typeof useLink>
