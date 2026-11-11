import { ref } from 'vue'

/** One line of the conversation: a question, a line about work, or an answer. */
export interface Turn {
  readonly id: string
  readonly voice: 'prompt' | 'doing' | 'response'
  readonly text: string
  readonly subject?: string
  readonly aside?: string
}

export interface ModelOption {
  readonly agentId: string
  readonly agentTitle: string
  readonly modelId: string
  readonly modelTitle: string
}

export const MODEL_OPTIONS: readonly ModelOption[] = [
  {
    agentId: 'claude',
    agentTitle: 'Claude',
    modelId: 'claude-3-7-sonnet',
    modelTitle: 'Sonnet 3.7',
  },
  { agentId: 'gemini', agentTitle: 'Gemini', modelId: 'gemini-2.5-pro', modelTitle: '2.5 Pro' },
]

const createAsked = (id: string, text: string): Turn => ({ id, voice: 'prompt', text })
const createAnswered = (id: string, text: string): Turn => ({ id, voice: 'response', text })
const createDid = (id: string, text: string, subject: string, aside: string): Turn => ({
  id,
  voice: 'doing',
  text,
  subject,
  aside,
})

const INITIAL_TURNS: readonly Turn[] = [
  createAsked('1', 'What does this note leave out?'),
  createDid('2', 'note neighborhood', 'Entropy', '4 links'),
  createAnswered(
    '3',
    'Two things. The note says the demon has to pay, but not where the payment ' +
      'lands — that is Landauer, and you have no note for him. And p. 373 counts ' +
      'arrangements at **fixed** energy, an assumption the free-energy note drops.',
  ),
  createAsked('4', 'Then make the Landauer note and put it under Entropy.'),
  createDid('5', 'note create', "Landauer's principle", 'under Entropy'),
  createAnswered(
    '6',
    'Made and filed under Entropy from p. 373 and the 1961 paper: what clearing ' +
      "one bit costs, and why the sorting pays it. Linked across to Maxwell's demon.",
  ),
]

/** The note the canned answer is about. */
const FOCUS_TITLE = 'Entropy'

/** How long the agent takes to answer, in milliseconds. */
const ANSWER_DELAY = 450

/** The conversation of the agent tab. The agent answers one canned line; the demo has no model behind it. */
export function createAgent() {
  const turns = ref<readonly Turn[]>(INITIAL_TURNS)
  const draft = ref('')
  const isWorking = ref(false)
  const selected = ref(MODEL_OPTIONS[0] as ModelOption)

  function submit(prompt: string) {
    const text = prompt.trim()
    if (!text || isWorking.value) return
    const title = FOCUS_TITLE
    const id = String(turns.value.length + 1)
    draft.value = ''
    turns.value = [
      ...turns.value,
      createAsked(`${id}-q`, text),
      createDid(`${id}-tool`, 'note search', title, '3 passages'),
    ]
    isWorking.value = true
    setTimeout(() => {
      turns.value = [
        ...turns.value,
        createAnswered(
          `${id}-a`,
          `In **${title}**, Boltzmann's count of microstates ($S = k_B \\ln W$) and ` +
            `Landauer's erasure bound ($k_B T \\ln 2$ per bit) describe the same constraint from ` +
            `two sides. *(Interactive browser demo — download Numen to run local models and agents on your own vault.)*`,
        ),
      ]
      isWorking.value = false
    }, ANSWER_DELAY)
  }

  const setDraft = (text: string) => {
    draft.value = text
  }

  const select = (modelId: string) => {
    selected.value = MODEL_OPTIONS.find((one) => one.modelId === modelId) ?? selected.value
  }

  return { turns, draft, isWorking, selected, submit, select, setDraft }
}

export type Agent = ReturnType<typeof createAgent>
