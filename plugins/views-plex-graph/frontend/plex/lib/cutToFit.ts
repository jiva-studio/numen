/** The one character a cut text ends in. */
const ELLIPSIS = '…'

const segmenter = new Intl.Segmenter(undefined, { granularity: 'grapheme' })

/**
 * The longest prefix of the label that fits the room, including the ellipsis, cut between
 * characters as a reader sees them: an emoji or a flag is kept whole or dropped.
 * The prefix is found by bisection, and a room with no space for any text returns the ellipsis alone.
 */
export function cutToFit(label: string, room: number, width: (label: string) => number): string {
  if (width(label) <= room) return label

  const characters = Array.from(segmenter.segment(label), (part) => part.segment)
  const trimTo = (count: number) => `${characters.slice(0, count).join('').trimEnd()}${ELLIPSIS}`

  let fits = 0
  let overCount = characters.length
  while (fits + 1 < overCount) {
    const middle = Math.floor((fits + overCount) / 2)
    if (width(trimTo(middle)) <= room) fits = middle
    else overCount = middle
  }
  return trimTo(fits)
}
