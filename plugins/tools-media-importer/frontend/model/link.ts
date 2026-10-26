/**
 * One link as its tab shows it: the address, the text fetched for it and the copy kept of its
 * video, read again whenever the work on it ends.
 */
import { computed, ref, shallowRef } from 'vue'
import { splitParagraphs } from '../lib/text'
import type { Links, LinkSource } from '../types'

export function useLink(links: Links, path: string) {
  const url = ref('')
  const text = ref('')
  const source = shallowRef<LinkSource | null>(null)
  const isCopyable = ref(false)
  const isWorking = ref(false)
  const isLoaded = ref(false)
  const error = ref('')
  let turn = 0

  const load = async (): Promise<void> => {
    const mine = ++turn
    try {
      const facts = await links.getLink(path)
      if (mine !== turn) return
      if (!facts.ok) {
        error.value = facts.error
        return
      }
      const [article, played] = await Promise.all([
        links.readArticle(path),
        facts.value.copySize > 0 ? links.getSource(path) : Promise.resolve(null),
      ])
      if (mine !== turn) return
      url.value = facts.value.url
      isCopyable.value = facts.value.isCopyable
      text.value = article
      source.value = played
      error.value = ''
    } catch (failure) {
      if (mine === turn) error.value = failure instanceof Error ? failure.message : String(failure)
    } finally {
      if (mine === turn) isLoaded.value = true
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
    isCopyable,
    isWorking,
    isLoaded,
    error,
    close: stop,
  }
}

export type LinkState = ReturnType<typeof useLink>
