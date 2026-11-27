import type { Row } from '@numen/ui'

/** The vault the window is drawn over: notes on entropy, the papers they cite, two lectures and two decks. */
export const VAULT_NAME = 'Physics'

const createFileRow = (path: string): Row => ({
  id: path,
  name: path.split('/').pop() ?? path,
  hasChildren: false,
})

const createFolderRow = (path: string, names: readonly string[]): Row => ({
  id: path,
  name: path,
  hasChildren: true,
  rows: names.map((name) => createFileRow(`${path}/${name}`)),
})

export const ROWS: readonly Row[] = [
  createFolderRow('Physics', [
    'Carnot cycle.md',
    'Entropy.md',
    'Entropy of mixing.md',
    'Fluctuation theorem.md',
    'Free energy.md',
    'Heat death.md',
    "Maxwell's demon.md",
    "Szilard's engine.md",
    'The second law.md',
  ]),
  createFolderRow('Computation', [
    'Algorithmic entropy.md',
    'Error-correcting codes.md',
    "Landauer's principle.md",
    'Reversible computing.md',
    'Shannon entropy.md',
  ]),
  createFolderRow('Reading', ['Boltzmann 1877.pdf', 'On Heat and Order.epub', 'Shannon 1948.pdf']),
  createFolderRow('Lectures', ['Lecture 4.mp3', 'Lecture 5.mp3']),
  createFolderRow('Decks', ['Sanskrit nouns.md', 'Sanskrit roots.md']),
  createFileRow('Inbox.md'),
  createFileRow('Reading list.md'),
]

/** The folders drawn open; the folder of the chosen row is opened as well. */
export const OPEN = ['Physics', 'Computation', 'Reading']

/** The texts of the notes a pane of the window can show, by their path in the vault. */
export const NOTES = {
  'Physics/Entropy.md': `How many arrangements a state could have been made of, counted the same way in a gas and in a message. Boltzmann counted the microstates a temperature leaves open; Shannon counted the messages a channel could have sent, and the two counts came out as one expression.

## Where the two meet

A demon that sorts fast molecules from slow ones appears to lower the entropy of the gas for nothing. It does not: sorting means knowing which molecule is which, and the knowing is paid for when the memory holding it is cleared.

- Boltzmann, 1877 — *Über die Beziehung*, p. 373
- Shannon, 1948 — the same logarithm, in bits
- Landauer, 1961 — what clearing one bit costs

> The measure is not about heat. It is about how much a description leaves open.

## What is still open

Whether Free energy belongs under this note or beside it. The counting there assumes a temperature the surroundings hold fixed, and that assumption is nowhere above.
`,
  'Physics/The second law.md': `Entropy never falls in a closed system, which is the whole of it stated in one line. Everything else here is about why the line holds, and how rarely it could fail.

## Boltzmann's count

The law is not a force that pushes a gas toward disorder. There are more ways to be spread out than to be gathered, and a gas wanders among its ways at random. From [Boltzmann 1877, p. 373](Reading/Boltzmann%201877.pdf#page=373):

> The number of these arrangements is not a property of any one of them. It belongs to the description: it says how much that description has left open.

## What follows from it

- Heat flows from hot to cold, because that direction has more arrangements
- No engine turns all of its heat into work
- A sorted gas left alone does not stay sorted

## What is still open

Why the past had so few arrangements. The law gives the arrow of time only once a low-entropy start is granted.
`,
  "Physics/Maxwell's demon.md": `A being at a door between two chambers of gas, opening it for the fast molecules going one way and the slow ones going the other. Maxwell set it up in 1867 to show that the second law holds only on average.

## Lecture 4

[Lecture 4](Lectures/Lecture%204.mp3) leaves the demon at the door and sends the bill to its memory: at 0:45, clearing the record of one molecule costs kT ln 2.

- Maxwell, 1867 — the door and the being, in a letter to Tait
- Szilard, 1929 — one molecule, one bit
- Bennett, 1982 — measuring can be free; forgetting cannot

> The demon does not cheat the law. It moves the cost into its memory.

## For the seminar

Does a demon with an endless memory escape? Bennett says it only postpones the bill: the memory itself becomes the disordered system.
`,
  "Computation/Landauer's principle.md": `Clearing one bit of memory costs at least kT ln 2 of heat, which is what the sorting has to pay for. At room temperature that is about 3 × 10⁻²¹ joules: small, but never zero.

## Why erasing, and not computing

Most steps of a computation can be run backwards. Erasing cannot: two different memories become the same blank one, and the arrangements that told them apart leave as heat.

- Landauer, 1961 — *Irreversibility and Heat Generation in the Computing Process*
- Bennett, 1973 — logically reversible machines
- Bérut et al., 2012 — the bound measured with one bead in a laser trap

> Information is physical. It is written in something, and it is wiped out of something.

## For the demon

The demon's memory fills one bit per molecule. Clearing that memory gives back all the entropy the sorting took from the gas.
`,
  'Computation/Shannon entropy.md': `The average number of yes-or-no questions a message leaves open. A fair coin leaves one; a coin that always lands heads leaves none. Shannon wrote it as H = −Σ p log p and called it entropy.

## Entropy as missing information

The count is the one Boltzmann made, in bits instead of joules per kelvin. A message a channel could have sent is an arrangement a gas could have been in.

- Shannon, 1948 — *A Mathematical Theory of Communication*
- Jaynes, 1957 — statistical mechanics read as inference
- one bit is k ln 2 of thermodynamic entropy

> A message tells as much as it was unlikely to be sent.

## Where it stops

The measure needs the probabilities given in advance. It says how much a message could tell, never what it means.
`,
  'Computation/Reversible computing.md': `A computation that keeps enough of its past to run backwards erases nothing, and so owes nothing under Landauer's bound. Bennett showed in 1973 that any computation can be made so, at the price of memory.

## The trick

Compute the answer, copy it out, then run every step backwards until the scratch memory is blank again. Only the copy is new, and copying onto a blank bit costs nothing.

- Toffoli gate — reversible, and enough for any logic
- Fredkin's billiard-ball computer — collisions as gates
- adiabatic circuits — the engineering version, slow but cool

> What you do not erase, you do not pay for.

## The catch

Staying reversible means running slowly. Every real machine still erases, and still heats.
`,
  'Computation/Error-correcting codes.md': `Redundancy added on purpose, so that a message can arrive through noise and still be read. Shannon proved that below the capacity of a channel the errors can be made as rare as one likes.

## Hamming (7,4)

Four bits of message, three of parity. Any single flipped bit can be found and turned back.

- parity checks over overlapping groups of bits
- the syndrome names the position of the error
- two errors at once are seen but not mended

> Noise is not the enemy of a message. Too little redundancy is.

## The cost in heat

Correcting an error means forgetting the wrong bit. Every correction pays the Landauer price, and a memory kept clean by correction is a small refrigerator.
`,
  'Reading list.md': `Papers to read before the exam, oldest first. Each one gets a note in Physics or Computation once it is read.

## Before the exam

- Maxwell, 1867 — the letter to Tait
- Szilard, 1929 — *Über die Entropieverminderung*
- Landauer, 1961 — nine pages
- Read Bennett 1982 by 19 April
- Jaynes, 1957 — information theory and statistical mechanics

## Later

- Zurek, 1989 — algorithmic randomness and physical entropy
- Bérut et al., 2012 — the experiment

> Read the old papers first. The new ones assume you have.
`,
  "Physics/Szilard's engine.md": `One molecule in a box, a partition slid in at the middle, and one question: which side is it on? Knowing the answer lets the molecule push the partition and lift a weight, kT ln 2 of work from a single bit.

## The cycle

- slide the partition in
- measure which side holds the molecule
- let it push a piston outward
- take the partition out and start again

> One bit of knowledge is worth kT ln 2 of work, and costs as much to forget.

## Where the bill went

Szilard put the cost in the measurement. Landauer and Bennett moved it to the reset of the memory that holds the answer.
`,
  'Computation/Algorithmic entropy.md': `The length of the shortest program that prints a string. A million zeros have a short program; a million coin flips have none shorter than themselves.

## Beside Shannon

Shannon's entropy belongs to a source of messages; this one belongs to a single message. For long typical messages the two agree on average.

- Kolmogorov, 1965 — three approaches to the quantity of information
- Chaitin, 1966 — the same idea, found separately
- Zurek, 1989 — physical entropy as missing data plus the length of the record

> A pattern is a description shorter than the thing described.

## What is still open

The measure cannot be computed, only bounded from above. Whether that matters for physics is the question Zurek leaves open.
`,
  'Physics/Fluctuation theorem.md': `The second law holds on average, and this says how often it fails. A small system can lose entropy for a moment; the odds against it grow exponentially with the amount lost.

## The ratio

P(+σ) / P(−σ) = e^σ for the entropy σ produced over a stretch of time: the chance of producing it against the chance of absorbing it.

- Evans, Cohen and Morriss, 1993 — first seen in simulation
- Crooks, 1999 — the same for the work done on a system
- Wang et al., 2002 — a bead in water, briefly running backwards

> The law is not broken by small things. It is only enforced on large ones.

## For the demon

A demon needs no trick to see entropy fall for a few molecules. It needs one to keep the fall.
`,
  'Physics/Heat death.md': `The state a closed universe tends toward: one temperature everywhere, nothing left to flow, no work to be had. Kelvin drew the conclusion in 1852 from Carnot's engines.

## Why nothing happens there

Work needs a difference. An engine runs between a hot body and a cold one; when every body has the same temperature, every engine stops.

- Kelvin, 1852 — the dissipation of mechanical energy
- Clausius, 1865 — the entropy of the world tends to a maximum
- Eddington, 1928 — the arrow of time

> Every process uses up a little of the difference that let it happen.

## What is still open

Whether gravity changes the count. A gas spreads out, but a cloud of stars clumps, and for it clumping means more arrangements, not fewer.
`,
  'Physics/Carnot cycle.md': `Two isotherms and two adiabats: the most an engine can do between two temperatures. No engine running between the same two reservoirs turns a larger share of its heat into work.

## The bound

η = 1 − T_cold / T_hot. Nothing about steam, pistons or the working gas appears in it, only the two temperatures.

- expand at T_hot, taking heat in
- expand without heat, cooling to T_cold
- compress at T_cold, giving heat out
- compress without heat, back to T_hot

> Carnot found the second law before anyone had the first.

## Why it matters here

The cycle is reversible, so it produces no entropy. Clausius defined entropy as the quantity that comes back unchanged around it.
`,
  'Inbox.md': `Things to file. Anything here longer than a week goes into a note or goes away.

## This week

- Ask about the Gibbs paradox
- Is the demon's memory part of the gas? Lecture 4, 0:45
- Link Shannon 1948, p. 11 from Shannon entropy
- Find the Bérut paper for Landauer's principle

> If it is worth keeping, it is worth a title.

## Heard in passing

Someone said the entropy of mixing jumps when the two gases become identical. Check it against Entropy of mixing before believing it.
`,
  'Physics/Entropy of mixing.md': `Two gases, a partition between them, and the partition taken away. If the gases differ, the entropy rises by 2N k ln 2; if they are the same gas, nothing happens at all.

## The Gibbs paradox

The jump is sudden. However alike two gases are, mixing them gives the full amount, until they are the same, when it gives nothing.

- Gibbs, 1875 — the paradox as he stated it
- the N! in the count, which makes identical molecules one arrangement
- Jaynes, 1992 — the entropy depends on which differences you can use

> Entropy measures what we can tell apart, not what is there.

## What is still open

Whether an experimenter who cannot separate two isotopes should count them as one gas.
`,
  'Physics/Free energy.md': `The part of the energy of a system that can still be turned into work while the surroundings hold the temperature fixed. F = U − TS: the energy, less what the entropy keeps locked up.

## Derive the Helmholtz form

- start from dU = T dS − p dV
- subtract d(TS) from both sides
- read off dF = −S dT − p dV

> Energy is conserved. What runs out is the free part of it.

## Where this belongs

Under Entropy or beside it. The counting here assumes a temperature the surroundings hold fixed, and the Entropy note assumes nothing of the kind.
`,
  'Templates/Lecture notes.md': `# {{ .title }}

Notes on the lecture of {{ .now.Format "2 January 2006" }}, written the same evening.

## What was claimed

The one statement the lecture was built to reach, in a single sentence.

## How it got there

- the setup, with the system and what is held fixed
- the step that does the work
- the result, and what it costs

> The question I would have asked if there had been time.

## To read

The paper the lecture cites, with the page, so the claim can be checked.
`,
  'Information is physical.md': `A recorded talk on Landauer's principle and the demon, saved to watch before the exam. The captions are kept as the text of this note.

## Why it is here

The talk walks through Szilard's engine with one molecule, then shows the 2012 bead experiment that measured the cost of erasing a bit.

- 0:00 — the demon at the door
- 12:40 — one molecule, one bit
- 31:15 — the bead in the laser trap

> Information is not abstract. It sits in a memory, and the memory is hot.

## Follow-up

Compare the numbers at 31:15 with Landauer's principle.
`,
} as const satisfies Readonly<Record<string, string>>

export type NotePath = keyof typeof NOTES

/** The note a pane of the window shows when a frame names none. */
export const DEFAULT_NOTE: NotePath = 'Physics/Entropy.md'

/** Returns the title a tab gives the note at `path`: its file name without the extension. */
export const getNoteTitle = (path: string): string =>
  (path.split('/').pop() ?? path).replace(/\.md$/, '')
