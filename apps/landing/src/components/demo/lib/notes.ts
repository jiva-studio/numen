/** One heading of a note: the line it is on, its text and its level. */
interface NoteHeading {
  readonly line: number
  readonly text: string
  readonly level: number
}

/** One note of the demo vault. */
export interface VaultNote {
  path: string
  title: string
  folder: string
  body: string
  parents: string[]
  jumps: string[]
  isHiddenFromFolderList?: boolean
  headings?: readonly NoteHeading[]
}

export const ENTROPY_PATH = 'physics/Entropy.md'

export const INITIAL_FOLDERS: readonly string[] = ['physics', 'computation', 'reading', 'lectures', 'decks']

export const INITIAL_NOTES: readonly VaultNote[] = [
  {
    path: 'physics/Entropy.md',
    title: 'Entropy',
    folder: 'physics',
    parents: ['physics/Thermodynamics.md', 'computation/Information theory.md'],
    jumps: ["physics/Maxwell's demon.md", 'computation/Shannon entropy.md'],
    headings: [
      { line: 1, text: 'What it measures', level: 2 },
      { line: 3, text: 'Where the two meet', level: 2 },
      { line: 13, text: 'What is still open', level: 2 },
    ],
    body: `How many arrangements a state could have been made of, counted the same way in a gas and in a message. Boltzmann counted the microstates a temperature leaves open; Shannon counted the messages a channel could have sent, and the two counts came out as one expression.

## Where the two meet

A demon that sorts fast molecules from slow ones appears to lower the entropy of the gas for nothing. It does not: sorting means knowing which molecule is which, and the knowing is paid for when the memory holding it is cleared.

- Boltzmann, 1877 — *Über die Beziehung*, p. 373
- Shannon, 1948 — the same logarithm, in bits
- Landauer, 1961 — what clearing one bit costs

> The measure is not about heat. It is about how much a description leaves open.

## What is still open

Whether [[Free energy]] belongs under this note or beside it. The counting there assumes a temperature the surroundings hold fixed, and that assumption is nowhere above.
`,
  },
  {
    path: 'physics/Entropy of mixing.md',
    title: 'Entropy of mixing',
    folder: 'physics',
    parents: ['physics/Entropy.md'],
    jumps: ['physics/Free energy.md'],
    body: `When two ideal gases at the same temperature and pressure are allowed to mix by removing a partition, the total [[Entropy]] rises because each molecule now has the whole volume open to it.

## Gibbs paradox

If the two gases are identical, removing the partition changes nothing macroscopic. Counting arrangements requires dividing by $N!$ for indistinguishable particles.
`,
  },
  {
    path: 'physics/The second law.md',
    title: 'The second law',
    folder: 'physics',
    parents: ['physics/Entropy.md'],
    jumps: ['physics/Heat death.md'],
    headings: [
      { line: 3, text: 'As Clausius put it', level: 2 },
      { line: 7, text: 'As Kelvin put it', level: 2 },
      { line: 11, text: 'Why the two are one', level: 2 },
    ],
    body: `Entropy never falls in an isolated system, which is the whole law stated in one line.

## As Clausius put it

Heat can never pass from a colder to a warmer body without some other change, connected therewith, occurring at the same time.

## As Kelvin put it

No cyclic process can convert heat from a single reservoir entirely into mechanical work.

## Why the two are one

A system moves from macrostates with fewer microstates to macrostates with overwhelmingly more, because there are more of the latter to be reached.
`,
  },
  {
    path: 'physics/Free energy.md',
    title: 'Free energy',
    folder: 'physics',
    parents: ['physics/Entropy.md'],
    jumps: ["computation/Landauer's principle.md"],
    body: `When a system is held at fixed temperature $T$ by a heat bath, the quantity that falls at equilibrium is not energy alone and not negative [[Entropy]] alone, but the Helmholtz free energy:

$$F = U - T S$$

## Work extractable

The decrease $-\\Delta F$ bounds the maximum useful work that can be drawn from a system in thermal contact with its surroundings.
`,
  },
  {
    path: 'physics/Thermodynamics.md',
    title: 'Thermodynamics',
    folder: 'physics',
    parents: [],
    jumps: [],
    isHiddenFromFolderList: true,
    body: `The bookkeeping of heat, work, and the state variables of macroscopic matter.

## Laws and potentials

1. Conservation of total energy ($dU = \\delta Q - \\delta W$)
2. [[Entropy]] never decreases in an isolated system — [[The second law]]
3. [[Free energy]] measures the work extractable at constant temperature
`,
  },
  {
    path: 'computation/Information theory.md',
    title: 'Information theory',
    folder: 'computation',
    parents: [],
    jumps: ['reading/Shannon 1948.md'],
    isHiddenFromFolderList: true,
    body: `What a channel can carry, measured by the uncertainty removed when a message arrives.

## Core ideas

- [[Shannon entropy]] counts missing information in bits: $H(X) = -\\sum p(x) \\log_2 p(x)$
- [[Entropy]] in statistical mechanics and in coding share the same logarithmic form
- [[Landauer's principle]] ties logical erasure back to physical heat
`,
  },
  {
    path: "physics/Maxwell's demon.md",
    title: "Maxwell's demon",
    folder: 'physics',
    parents: ['physics/Thermodynamics.md'],
    jumps: ['physics/Entropy.md', "computation/Landauer's principle.md"],
    isHiddenFromFolderList: true,
    headings: [
      { line: 3, text: 'Why it looks free', level: 2 },
      { line: 7, text: 'Where the bill arrives', level: 2 },
    ],
    body: `A thought experiment of 1867: a being small enough to see single molecules stands at a door between two halves of a box, and lets the fast ones through one way and the slow ones the other.

## Why it looks free

Nothing is pushed and nothing is heated. The gas ends sorted, hot on one side and cold on the other, and no work appears to have been done to sort it.

## Where the bill arrives

The being has to know which molecule is which, and it has to hold that knowledge somewhere. The memory fills, and clearing it costs what [[Landauer's principle]] says a bit costs.

> The demon does not break the second law. It pays for the sorting in a currency nobody was counting.
`,
  },
  {
    path: 'computation/Shannon entropy.md',
    title: 'Shannon entropy',
    folder: 'computation',
    parents: ['computation/Information theory.md'],
    jumps: ['physics/Entropy.md', 'reading/Shannon 1948.md'],
    body: `The expected surprise of a discrete random variable, derived in 1948 as the unique measure satisfying continuity, monotonicity, and branching additivity.

## Entropy as missing information

For a source with distribution $P = (p_1, \\dots, p_n)$:

$$H(P) = -\\sum_{i=1}^n p_i \\log_2 p_i$$

When all outcomes are equally likely, $H$ reduces to $\\log_2 n$, matching Boltzmann's [[Entropy]] formula up to units.
`,
  },
  {
    path: "computation/Landauer's principle.md",
    title: "Landauer's principle",
    folder: 'computation',
    parents: ['physics/Entropy.md', 'computation/Information theory.md'],
    jumps: ["physics/Maxwell's demon.md", 'physics/Free energy.md'],
    body: `Clearing one bit of memory costs at least $k_B T \\ln 2$ of heat dissipated into the surroundings, which is what the sorting in [[Maxwell's demon]] has to pay for.

## Logical irreversibility

A 2-to-1 map such as \`RESET -> 0\` compresses two logical states into one. By Liouville's theorem, phase-space volume cannot shrink, so the compressed factor of 2 must be exported to the thermal bath as [[Entropy]].
`,
  },
  {
    path: "physics/Boltzmann's constant.md",
    title: "Boltzmann's constant",
    folder: 'physics',
    parents: ['physics/The second law.md'],
    jumps: [],
    isHiddenFromFolderList: true,
    body: `The bridge factor $k_B = 1.380649 \\times 10^{-23}\\,\\text{J/K}$ converting a pure count of arrangements ($\\ln W$) into thermodynamic units of energy per kelvin.
`,
  },
  {
    path: 'physics/Heat death.md',
    title: 'Heat death',
    folder: 'physics',
    parents: ['physics/The second law.md'],
    jumps: [],
    isHiddenFromFolderList: true,
    body: `The asymptotic state in which all gradients of temperature, pressure, and chemical potential have been spent, leaving maximum [[Entropy]] and no [[Free energy]] to drive further work.
`,
  },
  {
    path: 'Inbox.md',
    title: 'Inbox',
    folder: '',
    parents: [],
    jumps: [],
    body: `Quick notes waiting to be filed into the graph:

- Compare Jaynes' maximum-entropy principle with Gibbs ensembles
- Check page 373 of *Boltzmann 1877* on discrete energy cells
- Link [[Shannon entropy]] to channel capacity notes
`,
  },
  {
    path: 'Reading list.md',
    title: 'Reading list',
    folder: '',
    parents: ['physics/Thermodynamics.md'],
    jumps: ['reading/Shannon 1948.md'],
    body: `Primary sources in this vault:

1. **Boltzmann (1877)** — *Über die Beziehung zwischen dem zweiten Hauptsatze der mechanischen Wärmetheorie und der Wahrscheinlichkeitsrechnung*
2. **Shannon (1948)** — *A Mathematical Theory of Communication*
3. **Landauer (1961)** — *Irreversibility and Heat Generation in the Computing Process*
`,
  },
  {
    path: 'reading/Shannon 1948.md',
    title: 'Shannon 1948',
    folder: 'reading',
    parents: ['physics/Thermodynamics.md', 'computation/Information theory.md'],
    jumps: ['computation/Shannon entropy.md'],
    isHiddenFromFolderList: true,
    body: `Reading notes on Claude Shannon's *A Mathematical Theory of Communication* (Bell System Technical Journal, 1948).

See [[Shannon entropy]] and [[Information theory]].
`,
  },
]

/** The label drawn on a link between two notes, by `from->to`. */
export const INITIAL_EDGE_LABELS: Readonly<Record<string, string>> = {
  'physics/Thermodynamics.md->physics/Entropy.md': 'measures',
  'physics/Entropy.md->physics/The second law.md': 'states',
  'physics/Entropy.md->physics/Free energy.md': 'costs',
  'computation/Information theory.md->computation/Shannon entropy.md': 'defines',
  "physics/Maxwell's demon.md->physics/Entropy.md": 'tests',
  'computation/Shannon entropy.md->physics/Entropy.md': 'see also',
}
