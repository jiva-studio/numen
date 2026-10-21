/**
 * Which cards stand in the corner, what each of them has left to say, and when
 * each of them goes.
 *
 * Work appears once it has lasted and goes when the work does; something that
 * is so stands while it is so; something that happened stands to be read and
 * then goes, unless it is trouble, which stands until it is put away. Nothing
 * here draws anything.
 */
import {
  computed,
  nextTick,
  ref,
  shallowRef,
  watch,
  watchEffect,
  type ComputedRef,
  type Ref,
  type ShallowRef,
} from 'vue'
import {
  getArrivalTimes,
  getNoticeDuration,
  getFinishedNotices,
  getShownNotices,
} from '../lib/duration'
import { foldNotices } from '../lib/fold'
import { measureMovement, type Movement } from '../lib/movement'
import { getStillAway, getReadable, findTally, type Notice } from '../lib/notice'
import { getRemainingWord } from '../lib/tally'
import { useNoticeStack } from './stack'

/** A card standing now, which carries the way away under it. */
interface CardHandle {
  readonly way: HTMLElement | null
}

export interface NoticeCardsOptions {
  /** What the window has to say, in the order it is drawn. */
  readonly getNotices: () => readonly Notice[]
  /** The corner itself, which a pointer and the keyboard are followed over. */
  readonly stack: Readonly<ShallowRef<HTMLElement | null>>
  /** How long work runs before it is worth a card. */
  readonly getWait: () => number
  /** How many cards stand at once. */
  readonly getRoom: () => number
  /** What the moment is. */
  readonly getNow: () => number
  /** Whether nobody is looking. */
  readonly isHidden: () => boolean
  /** A card is finished with: read long enough, or dismissed. */
  readonly dismiss: (id: string) => void
}

export interface NoticeCardsState {
  /** Everything standing, whether or not it is folded away behind the rest. */
  readonly drawn: ComputedRef<readonly Notice[]>
  /** What is drawn, and how many stand behind it. */
  readonly folds: ComputedRef<{ shown: readonly Notice[]; over: number }>
  /** Whether a person has asked to see what is folded away behind the rest. */
  readonly isOpened: Ref<boolean>
  /** What a counting notice has left to run, in words. */
  readonly leftOn: (one: Notice) => string
  /** A card as it is drawn, held under the notice it stands for. */
  readonly holdCard: (id: string, card: unknown) => void
  /** A card put away by hand. */
  readonly dismissByHand: (id: string) => Promise<void>
  readonly onPointerOver: (event: PointerEvent) => void
  readonly onPointerOut: (event: PointerEvent) => void
  readonly onFocusIn: () => void
  readonly onFocusOut: (event: FocusEvent) => void
}

function useNoticeSampling(
  getNotices: () => readonly Notice[],
  now: Ref<number>,
  beat: () => void,
) {
  const moving = shallowRef<ReadonlyMap<string, Movement>>(new Map())
  const sample = (): void => {
    beat()
    moving.value = measureMovement(moving.value, getNotices(), now.value)
  }
  const leftOn = (one: Notice): string => {
    const tally = findTally(one)
    if (tally === undefined) return ''
    return getRemainingWord(tally.total - tally.done, moving.value.get(one.id)?.rate ?? 0)
  }
  return { sample, leftOn }
}

interface DismissalContext {
  readonly folds: ComputedRef<{ shown: readonly Notice[] }>
  readonly away: ShallowRef<ReadonlySet<string>>
  readonly forgotten: Set<string>
  readonly dismiss: (id: string) => void
}

function useNoticeDismissal(context: DismissalContext) {
  const cards = new Map<string, CardHandle>()

  const holdCard = (id: string, card: unknown): void => {
    if (card) cards.set(id, card as CardHandle)
    else cards.delete(id)
  }

  const dismissByHand = async (id: string): Promise<void> => {
    const at = context.folds.value.shown.findIndex((one) => one.id === id)
    const hadFocus = cards.get(id)?.way === document.activeElement
    context.forgotten.add(id)
    context.away.value = new Set([...context.away.value, id])
    context.dismiss(id)
    if (!hadFocus || at < 0) return
    await nextTick()
    const left = context.folds.value.shown
    const next = left[Math.min(at, left.length - 1)]
    if (next) cards.get(next.id)?.way?.focus()
  }

  return { holdCard, dismissByHand }
}

function watchNoticeTicker(
  sample: () => void,
  rendered: ComputedRef<readonly Notice[]>,
  options: { getNotices: () => readonly Notice[]; away: ShallowRef<ReadonlySet<string>> },
): void {
  const coming = computed(() => {
    const displayed = new Set(rendered.value.map((one) => one.id))
    return getReadable(options.getNotices()).some(
      (one) => one.stay !== 'read' && !options.away.value.has(one.id) && !displayed.has(one.id),
    )
  })
  const timing = computed(() =>
    rendered.value.some(
      (one) => one.stay === 'read' && getNoticeDuration(one.text, one.about) !== Infinity,
    ),
  )
  const counting = computed(() => rendered.value.some((one) => findTally(one) !== undefined))

  watchEffect((clean) => {
    if (!coming.value && !timing.value && !counting.value) return
    const tick = setInterval(sample, 250)
    clean(() => clearInterval(tick))
  })
}

interface NoticeLifecycleOptions {
  readonly getNotices: () => readonly Notice[]
  readonly arrived: ShallowRef<ReadonlyMap<string, number>>
  readonly away: ShallowRef<ReadonlySet<string>>
  readonly forgotten: Set<string>
  readonly read: ShallowRef<number>
  readonly sample: () => void
  readonly dismiss: (id: string) => void
}

function useNoticeLifecycle(options: NoticeLifecycleOptions): void {
  const { getNotices, arrived, away, forgotten, read, sample, dismiss } = options
  watch(
    getNotices,
    (all) => {
      sample()
      arrived.value = getArrivalTimes(arrived.value, all, read.value)
      away.value = getStillAway(away.value, all)
      const here = new Set(getReadable(all).map((one) => one.id))
      for (const id of [...forgotten]) if (!here.has(id)) forgotten.delete(id)
    },
    { immediate: true },
  )

  watchEffect(() => {
    for (const id of getFinishedNotices(getNotices(), arrived.value, read.value)) {
      if (forgotten.has(id)) continue
      forgotten.add(id)
      dismiss(id)
    }
  })
}

export function useNoticeCards(options: NoticeCardsOptions): NoticeCardsState {
  const away = shallowRef<ReadonlySet<string>>(new Set())
  const arrived = shallowRef<ReadonlyMap<string, number>>(new Map())
  const cleared = new Set<string>()

  const { now, read, beat, onPointerOver, onPointerOut, onFocusIn, onFocusOut } = useNoticeStack(
    options.stack,
    options.getNow,
    options.isHidden,
  )

  const { sample, leftOn } = useNoticeSampling(options.getNotices, now, beat)

  useNoticeLifecycle({
    getNotices: options.getNotices,
    arrived,
    away,
    forgotten: cleared,
    read,
    sample,
    dismiss: options.dismiss,
  })

  const rendered = computed(() =>
    getShownNotices(options.getNotices(), read.value, {
      firstSeen: arrived.value,
      away: away.value,
      wait: options.getWait(),
    }),
  )

  const isOpened = ref(false)
  const folds = computed(() =>
    foldNotices(rendered.value, isOpened.value ? rendered.value.length : options.getRoom()),
  )

  watch(rendered, (all) => {
    if (all.length <= options.getRoom()) isOpened.value = false
  })

  watchNoticeTicker(sample, rendered, { getNotices: options.getNotices, away })

  const { holdCard, dismissByHand } = useNoticeDismissal({
    folds,
    away,
    forgotten: cleared,
    dismiss: options.dismiss,
  })

  return {
    drawn: rendered,
    folds,
    isOpened,
    leftOn,
    holdCard,
    dismissByHand,
    onPointerOver,
    onPointerOut,
    onFocusIn,
    onFocusOut,
  }
}
