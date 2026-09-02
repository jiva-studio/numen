import type { SelectChoice } from '.'

/** A run of choices drawn together, under the name they share. */
export interface Shelf {
  /** What the shelf is called. A run of choices naming none carries none. */
  readonly label?: string
  readonly choices: readonly SelectChoice[]
}

/**
 * The choices in the runs they are drawn in: one run for each stretch of them
 * naming the same shelf, in the order they were offered.
 */
export const shelved = (choices: readonly SelectChoice[]): readonly Shelf[] => {
  const shelves: { label?: string; choices: SelectChoice[] }[] = []
  for (const choice of choices) {
    const last = shelves.at(-1)
    if (last && last.label === choice.group) last.choices.push(choice)
    else shelves.push({ ...(choice.group ? { label: choice.group } : {}), choices: [choice] })
  }
  return shelves
}
