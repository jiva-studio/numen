interface DeckField {
  readonly field: string
  readonly text: string
}

export interface DeckCard {
  readonly id: string
  readonly section: string | null
  readonly cardModel: string
  readonly filled: readonly DeckField[]
}

export interface DeckSection {
  readonly id: string
  readonly name: string
}

export const INITIAL_CARDS: readonly DeckCard[] = [
  {
    id: 'boltzmann-entropy',
    section: 'entropy',
    cardModel: 'Concept',
    filled: [
      { field: 'Concept', text: 'Boltzmann entropy' },
      { field: 'Formula', text: 'S = k_B ln W' },
      { field: 'Meaning', text: 'Logarithm of the microstates consistent with a macrostate' },
      { field: 'Source', text: 'Boltzmann, 1877 — p. 373' },
    ],
  },
  {
    id: 'shannon-entropy',
    section: 'entropy',
    cardModel: 'Concept',
    filled: [
      { field: 'Concept', text: 'Shannon entropy' },
      { field: 'Formula', text: 'H(X) = -∑ p(x) log₂ p(x)' },
      { field: 'Meaning', text: 'Expected bits of uncertainty removed when a message arrives' },
      { field: 'Source', text: 'Shannon, 1948 — § 6' },
    ],
  },
  {
    id: 'helmholtz-free-energy',
    section: 'entropy',
    cardModel: 'Concept',
    filled: [
      { field: 'Concept', text: 'Helmholtz free energy' },
      { field: 'Formula', text: 'F = U - T S' },
      { field: 'Meaning', text: 'Maximum work extractable at constant temperature T' },
      { field: 'Source', text: 'Thermodynamics' },
    ],
  },
  {
    id: 'landauer-bound',
    section: 'erasure',
    cardModel: 'Law',
    filled: [
      { field: 'Law', text: "Landauer's principle" },
      { field: 'Statement', text: 'Erasing 1 bit of information dissipates at least k_B T ln 2' },
      { field: 'Consequence', text: "Pays the entropy bill run up by Maxwell's demon" },
    ],
  },
  {
    id: 'second-law-card',
    section: 'erasure',
    cardModel: 'Law',
    filled: [
      { field: 'Law', text: 'Second law of thermodynamics' },
      { field: 'Statement', text: 'ΔS ≥ 0 in any isolated system' },
      { field: 'Consequence', text: 'No description improves its own precision for nothing' },
    ],
  },
]

export const INITIAL_SECTIONS: readonly DeckSection[] = [
  { id: 'entropy', name: 'Counting arrangements' },
  { id: 'erasure', name: 'Information & erasure' },
]

/** The preset that schedules the reviews of the deck. */
export const DECK_PRESET = 'Physics'
