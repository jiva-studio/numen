export type StepStarts = readonly (number | null)[]

export type StepPhase = 'done' | 'current' | 'ahead'

export type Segment = {
  readonly index: number
  readonly left: number
  readonly width: number
}

export const findStepAt = (starts: StepStarts, time: number): number => {
  let found = -1
  starts.forEach((start, index) => {
    if (start !== null && start <= time) found = index
  })
  return found
}

export const findNextStart = (starts: StepStarts, time: number): number | null => {
  const after = time + 0.05
  for (const start of starts) {
    if (start !== null && start > after) return start
  }
  return null
}

export const findPreviousStart = (starts: StepStarts, time: number): number => {
  const started = starts.filter((start): start is number => start !== null && start <= time)
  const current = started.at(-1)
  if (current === undefined) return 0
  if (time - current > 1) return current
  return started.at(-2) ?? 0
}

export const getStepPhases = (count: number, currentIndex: number): StepPhase[] =>
  Array.from({ length: count }, (_, index) => {
    if (index < currentIndex) return 'done'
    return index === currentIndex ? 'current' : 'ahead'
  })

export const getSegments = (starts: StepStarts, length: number): Segment[] => {
  if (!(length > 0)) return []
  const ran = starts.flatMap((start, index) => (start === null ? [] : [{ index, start }]))
  return ran.flatMap(({ index, start }, position) => {
    const left = Math.min((start * 100) / length, 100)
    const end = Math.min(((ran[position + 1]?.start ?? length) * 100) / length, 100)
    return end > left ? [{ index, left, width: end - left }] : []
  })
}

export const getVideoLength = (
  mediaDuration: number,
  steps: readonly { readonly start: number | null; readonly duration: number }[],
): number => {
  if (Number.isFinite(mediaDuration) && mediaDuration > 0) return mediaDuration
  return steps.reduce(
    (end, step) => (step.start === null ? end : Math.max(end, step.start + step.duration / 1000)),
    0,
  )
}

export const getSetupLength = (steps: readonly { readonly keyword: string }[]): number => {
  const index = steps.findIndex((step) => step.keyword === 'When' || step.keyword === 'Then')
  return index === -1 ? steps.length : index
}

export const findActionStart = (starts: StepStarts, setupLength: number): number | null =>
  starts.slice(setupLength).find((start) => start !== null) ?? null

export type Input = {
  readonly kind: 'down' | 'move' | 'up' | 'key' | 'type'
  readonly t: number
  readonly x: number
  readonly y: number
  readonly w?: number
  readonly h?: number
  readonly key?: string
  readonly mods?: readonly string[]
  readonly isField?: boolean
  readonly text?: string
  readonly isAppend?: boolean
}

export type Ripple = { readonly x: number; readonly y: number; readonly progress: number }

export type Cursor = {
  readonly x: number
  readonly y: number
  readonly ripple: Ripple | null
  readonly isPressed: boolean
}

export type Badge = { readonly parts: readonly string[]; readonly opacity: number }

export type Highlight = {
  readonly x: number
  readonly y: number
  readonly w: number
  readonly h: number
  readonly opacity: number
  readonly index: number
}

export const getCursorAt = (inputs: readonly Input[], time: number): Cursor | null => {
  const travelLength = 0.6
  const rippleLength = 0.4
  const convertShare = (share: number): number =>
    share < 0.5 ? 2 * share * share : 1 - (-2 * share + 2) ** 2 / 2
  const clamp = (share: number): number => Math.min(Math.max(share, 0), 1)
  const isPressing = (input: Input): boolean => input.kind === 'down' || input.kind === 'move'
  const isDragging = (from: Input, to: Input): boolean =>
    isPressing(from) && (to.kind === 'move' || to.kind === 'up')
  const getShare = (from: Input, to: Input): number => {
    const start = isDragging(from, to) ? from.t : Math.max(from.t, to.t - travelLength)
    if (to.t <= start) return 1
    const share = clamp((time - start) / (to.t - start))
    return isDragging(from, to) ? share : convertShare(share)
  }
  const getRipple = (points: readonly Input[]): Ripple | null => {
    const down = points.filter((point) => point.kind === 'down' && point.t <= time).at(-1)
    if (!down || time - down.t >= rippleLength) return null
    return { x: down.x, y: down.y, progress: (time - down.t) / rippleLength }
  }
  const points = inputs.filter(
    (input) => input.kind === 'down' || input.kind === 'move' || input.kind === 'up',
  )
  const first = points.at(0)
  if (!first) return null
  const path: Input[] = [{ kind: 'up', t: Math.min(first.t, 0), x: 0.5, y: 0.5 }, ...points]
  const nextIndex = path.findIndex((point) => point.t > time)
  const previous = path.at(nextIndex === -1 ? -1 : Math.max(nextIndex - 1, 0)) ?? first
  const next = nextIndex > 0 ? path.at(nextIndex) : undefined
  const share = next ? getShare(previous, next) : 0
  const to = next ?? previous
  return {
    x: previous.x + (to.x - previous.x) * share,
    y: previous.y + (to.y - previous.y) * share,
    ripple: getRipple(points),
    isPressed: isPressing(previous),
  }
}

export const getBadgeAt = (inputs: readonly Input[], time: number): Badge | null => {
  const groupGap = 0.6
  const showLength = 1
  const fadeLength = 0.3
  const textLength = 40
  const partCount = 5
  const symbols: Record<string, string> = { Control: '⌃', Alt: '⌥', Shift: '⇧', Meta: '⌘' }
  const names: Record<string, string> = {
    Enter: 'Enter',
    Escape: 'Esc',
    ArrowDown: '↓',
    ArrowUp: '↑',
    ArrowLeft: '←',
    ArrowRight: '→',
    Backspace: '⌫',
    Delete: 'Del',
    ' ': 'Space',
  }
  const getMods = (input: Input): readonly string[] => input.mods ?? []
  const isCharacter = (input: Input): boolean =>
    (input.key ?? '').length === 1 && getMods(input).every((mod) => mod === 'Shift')
  const isShown = (input: Input): boolean => {
    if (input.t > time) return false
    if (input.kind === 'type') return !(input.isAppend && !input.text)
    return input.kind === 'key' && !(input.isField && isCharacter(input))
  }
  const getKeyName = (key: string): string =>
    names[key] ?? (key.length === 1 ? key.toUpperCase() : key)
  const getKeyLabel = (input: Input): string => {
    const mods = Object.keys(symbols).filter(
      (mod) => getMods(input).includes(mod) && !(mod === 'Shift' && isCharacter(input)),
    )
    return [...mods.map((mod) => symbols[mod]), getKeyName(input.key ?? '')].join(' ')
  }
  const truncateText = (text: string): string =>
    text.length > textLength ? `…${text.slice(1 - textLength)}` : text
  const findGroup = (shown: readonly Input[]): readonly Input[] => {
    let start = shown.length - 1
    const getGap = (index: number): number =>
      (shown.at(index)?.t ?? 0) - (shown.at(index - 1)?.t ?? 0)
    while (start > 0 && getGap(start) <= groupGap) start -= 1
    return shown.slice(start)
  }
  const collectParts = (group: readonly Input[]): string[] => {
    const parts: { typed: string | null; label: string }[] = []
    for (const input of group) {
      const last = parts.at(-1)
      if (input.kind !== 'type') parts.push({ typed: null, label: getKeyLabel(input) })
      else if (typeof last?.typed === 'string')
        last.typed = (input.isAppend ? last.typed : '') + (input.text ?? '')
      else parts.push({ typed: input.text ?? '', label: '' })
    }
    return parts.map((part) => (part.typed === null ? part.label : `Typing “${truncateText(part.typed)}”`))
  }
  const shown = inputs.filter(isShown)
  const last = shown.at(-1)
  if (!last || time - last.t >= showLength) return null
  const age = time - last.t
  const opacity = Math.min((showLength - age) / fadeLength, 1)
  return { parts: collectParts(findGroup(shown)).slice(-partCount), opacity }
}

export const getHighlightsAt = (inputs: readonly Input[], time: number): Highlight[] => {
  const showLength = 1
  const fadeLength = 0.3
  const replaceLength = 0.15
  const hasRect = (input: Input): boolean => (input.w ?? 0) > 0 && (input.h ?? 0) > 0
  const isSamePlace = (a: Input, b: Input): boolean =>
    a.x === b.x && a.y === b.y && a.w === b.w && a.h === b.h
  const groups: { first: Input; last: Input; index: number }[] = []
  inputs.forEach((input, index) => {
    if (input.kind !== 'key' && input.kind !== 'type') return
    if (!hasRect(input) || input.t > time) return
    const group = groups.at(-1)
    if (group && isSamePlace(group.last, input)) Object.assign(group, { last: input, index })
    else groups.push({ first: input, last: input, index })
  })
  return groups.flatMap((group, position) => {
    const age = time - group.last.t
    const next = groups.at(position + 1)
    const shown = Math.min((showLength - age) / fadeLength, 1)
    const kept = next ? 1 - (time - next.first.t) / replaceLength : 1
    const opacity = Math.min(shown, kept)
    if (opacity <= 0) return []
    const { x, y } = group.last
    return [{ x, y, w: group.last.w ?? 0, h: group.last.h ?? 0, opacity, index: group.index }]
  })
}
