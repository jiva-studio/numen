/** A paper of the demo: its title, how many leaves it has, the page it opens on and the lines of its sample pages. */
export interface PdfDoc {
  title: string
  leaves: number
  firstPage: number
  pages: readonly (readonly string[])[]
}

export const BOLTZMANN_PATH = 'reading/Boltzmann 1877.pdf'
export const SHANNON_PATH = 'reading/Shannon 1948.pdf'

export const PDF_DOCS: Readonly<Record<typeof BOLTZMANN_PATH | typeof SHANNON_PATH, PdfDoc>> = {
  'reading/Boltzmann 1877.pdf': {
    title: 'Boltzmann 1877',
    leaves: 412,
    firstPage: 372,
    pages: [
      [
        '§ 6.  THE COUNT OF ARRANGEMENTS',
        '',
        'Let a quantity of gas be enclosed, and let the total energy',
        'it holds be fixed. The molecules may then be arranged in a',
        'great many ways without that total being disturbed, and no',
        'measurement made upon the gas as a whole distinguishes one',
        'such arrangement from another.',
        '',
        'The number of these arrangements is not a property of any',
        'one of them. It belongs to the description: it says how much',
        'that description has left open. Where the description is',
        'exact, the number is one, and its logarithm vanishes.',
        '',
        'It is this logarithm, and not the heat, that the second law',
        'is a statement about. A quantity of gas passes from an',
        'arrangement that could have been reached in few ways to one',
        'that could have been reached in many, because there are more',
        'of the latter to be reached.',
      ],
      [
        'The same counting serves wherever a description leaves some-',
        'thing open. A channel that may carry any of a great number of',
        'messages is described by the logarithm of that number, and',
        'the description is improved exactly as the number falls.',
        '',
        'Nothing in the argument is peculiar to molecules. What is',
        'counted are the possibilities a statement admits, and the',
        'measure of them is the same whether the statement is made',
        'about a gas, about a message, or about the contents of a',
        'memory that is about to be cleared.',
      ],
      [
        'The objection does not hold. To open the door for the swift',
        'ones the observer must know which are swift, and the knowing',
        'occupies a memory. That memory is finite; to go on sorting,',
        'the observer must clear it.',
        '',
        'The clearing is the payment. It is exacted not at the door',
        'but at the moment the record of a molecule is destroyed, and',
        'it is exacted at the same rate at which the sorting reduced',
        'the count. The books balance, and the law stands.',
      ],
    ],
  },
  'reading/Shannon 1948.pdf': {
    title: 'Shannon 1948',
    leaves: 55,
    firstPage: 10,
    pages: [
      [
        '§ 6.  CHOICE, UNCERTAINTY AND ENTROPY',
        '',
        'Suppose we have a set of possible events whose probabilities',
        'of occurrence are p_1, p_2, ..., p_n. These probabilities are',
        'known but that is all we know concerning which event will',
        'occur. Can we find a measure of how much "choice" is involved',
        'in the selection of the event or of how uncertain we are of',
        'the outcome?',
        '',
        'If there is such a measure, say H(p_1, p_2, ..., p_n), it is',
        'reasonable to require of it that H be continuous in the p_i,',
        'and if all the p_i are equal, p_i = 1/n, then H should be a',
        'monotonic increasing function of n.',
        '',
        'Theorem 2: The only H satisfying the three assumptions is of',
        'the form H = -K sum(p_i log p_i), where K is a positive',
        'constant that merely amounts to a choice of a unit of measure.',
      ],
      [
        'Quantities of the form H = -sum(p_i log p_i) play a central',
        'role in information theory as measures of information, choice',
        'and uncertainty. The form of H will be recognized as that of',
        'entropy as defined in certain formulations of statistical',
        'mechanics where p_i is the probability of a system being in',
        'cell i of its phase space.',
        '',
        "H is then, for example, the H in Boltzmann's famous H-theorem.",
        'We shall call H = -sum(p_i log p_i) the entropy of the set of',
        'probabilities p_1, ..., p_n.',
      ],
    ],
  },
}
