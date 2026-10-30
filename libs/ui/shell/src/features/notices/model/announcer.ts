/**
 * What is read out of the corner, and what is read out over whatever else is
 * being read.
 *
 * Both regions are emptied and filled again a tick later. Two changes can land
 * inside one tick, and the second reads out everything neither has read yet.
 */
import { nextTick, onMounted, ref, shallowRef, watch, type Ref } from 'vue'
import type { Notice } from '../lib/notice'

/** What each card reads out as. */
const getWords = (one: Notice): string => (one.about ? `${one.text} — ${one.about}` : one.text)

/** The two regions the corner is read out through. */
export interface Announcer {
  /** What is read out politely, and what is read out over everything. */
  readonly polite: Ref<string>
  readonly assertive: Ref<string>
}

/**
 * The cards drawn now are read out as they arrive and as their words change.
 * Nothing is read out until the regions themselves have been drawn.
 */
export function useAnnouncer(getNotices: () => readonly Notice[]): Announcer {
  const polite = ref('')
  const assertive = ref('')

  /** The words each card was last read out by. */
  const announced = shallowRef<ReadonlyMap<string, string>>(new Map())
  /** Whether both regions have stood empty, which is what makes them read. */
  let isListening = false
  /** Which reading is the one in hand, and what is waiting to be read out. */
  let activeReadCount = 0
  let pendingNotices: readonly Notice[] = []

  const readOut = async (all: readonly Notice[]): Promise<void> => {
    if (!isListening) return
    const unannounced = all.filter((one) => announced.value.get(one.id) !== getWords(one))
    announced.value = new Map(all.map((one) => [one.id, getWords(one)]))
    if (unannounced.length === 0) return
    pendingNotices = [...pendingNotices, ...unannounced]

    const mine = ++activeReadCount
    polite.value = ''
    assertive.value = ''
    await nextTick()
    if (mine !== activeReadCount) return

    const pending = pendingNotices
    pendingNotices = []
    const loud = pending.filter((one) => one.severity === 'error')
    const regular = pending.filter((one) => one.severity !== 'error')
    if (loud.length) assertive.value = loud.map(getWords).join('. ')
    if (regular.length) polite.value = regular.map(getWords).join('. ')
  }

  watch(getNotices, async (all) => {
    await readOut(all)
  })

  onMounted(async () => {
    await nextTick()
    isListening = true
    await readOut(getNotices())
  })

  return { polite, assertive }
}
