import { describe, expect, it } from 'vitest'
import { effectScope } from 'vue'
import type { AgentPort, AgentStep, ConversationContext } from '@numen/plugin-sdk'
import { WORDS } from '../words'
import { useConversationView } from './useConversationView'

/** A port that keeps what it was questions and what it was told is over, and says what a test told it to. */
const createAgent = (says: AgentStep[] = [{ kind: 'said', text: 'Because of the leaves.' }]) => {
  const questions: { question: string; focus: string; conversation: string }[] = []
  const finished: string[] = []
  const agent: AgentPort = {
    async *ask(query) {
      questions.push({
        question: query.question,
        focus: query.focus,
        conversation: query.conversation,
      })
      for (const step of says) yield step
    },
    async finish(conversation) {
      finished.push(conversation)
    },
  }
  return { agent, questions, finished }
}

/** The view's state in a scope of its own, which a test disposes as the host unmounts the view. */
const open = (context: Partial<ConversationContext> = {}) => {
  const port = createAgent()
  const scope = effectScope()
  const view = scope.run(() =>
    useConversationView({ agent: port.agent, focus: 'decks/Words.md', ...context }, (draw) =>
      draw(),
    ),
  )!
  return { view, scope, ...port }
}

describe('the conversation view', () => {
  // The focus and the port carry what the questions are about; a name written into the question
  // is read as instruction.
  it('sends only the text typed, with the focus as a separate field', async () => {
    const { view, questions } = open()

    await view.send('why is it called that')

    expect(questions).toStrictEqual([
      {
        question: 'why is it called that',
        focus: 'decks/Words.md',
        conversation: questions[0]?.conversation,
      },
    ])
  })

  it('answers every question in one conversation', async () => {
    const { view, questions } = open()
    await view.send('one')
    await view.send('two')
    expect(questions[0]?.conversation).toBe(questions[1]?.conversation)
  })

  it('never gives one name to two views', async () => {
    const first = open()
    const second = open()
    await first.view.send('why')
    await second.view.send('why')
    expect(first.questions[0]?.conversation).not.toBe(second.questions[0]?.conversation)
  })

  it('ends the conversation when the view is taken away', async () => {
    const { view, scope, questions, finished } = open()
    await view.send('why')

    scope.stop()

    expect(finished).toStrictEqual([questions[0]?.conversation])
  })

  it('sends nothing when nothing was written', async () => {
    const { view, questions } = open()
    view.setQuestion('wh')

    await view.send('')

    expect(questions).toStrictEqual([])
    expect(view.question.value).toBe('wh')
  })

  it('clears what was written once it is sent', async () => {
    const { view } = open()
    view.setQuestion('why')

    await view.send('why')

    expect(view.question.value).toBe('')
    expect(view.turns.value.map((turn) => turn.text)).toContain('Because of the leaves.')
  })

  it('shows why the agent cannot be reached where the answers stand, and the silence otherwise', () => {
    let unreachable = 'The agent could not be reached.'
    const { view } = open({ getUnreachable: () => unreachable })
    expect(view.getSilence()).toBe('The agent could not be reached.')

    unreachable = ''
    expect(view.getSilence()).toBe(WORDS.nothingSaid)
  })

  it('names the field as the caller says, and as the vault chat does where it says nothing', () => {
    expect(open({ placeholder: 'Ask about this card' }).view.placeholder).toBe(
      'Ask about this card',
    )
    expect(open().view.placeholder).toBe(WORDS.ask)
  })
})

describe('a question in flight', () => {
  /** A port whose answer does not arrive until the test lets it. */
  const createSlowAgent = () => {
    let answer = () => {}
    const agent: AgentPort = {
      async *ask(query) {
        await new Promise<void>((then) => {
          answer = then
          query.signal.addEventListener('abort', () => then())
        })
        yield { kind: 'said', text: 'Because of the leaves.' }
      },
      async finish() {},
    }
    return { agent, answer: () => answer() }
  }

  it('is working while an answer is on its way, and not before or after', async () => {
    const slow = createSlowAgent()
    const view = effectScope().run(() =>
      useConversationView({ agent: slow.agent, focus: '' }, (draw) => draw()),
    )!
    expect(view.isWorking.value).toBe(false)

    const reply = view.send('why')
    expect(view.isWorking.value).toBe(true)

    slow.answer()
    await reply
    expect(view.isWorking.value).toBe(false)
  })

  it('stops working when the answer on its way is stopped', async () => {
    const slow = createSlowAgent()
    const view = effectScope().run(() =>
      useConversationView({ agent: slow.agent, focus: '' }, (draw) => draw()),
    )!
    const reply = view.send('why')

    view.stop()

    expect(view.isWorking.value).toBe(false)
    await reply
  })
})
