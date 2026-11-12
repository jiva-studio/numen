import { describe, expect, it } from 'vitest'
import { effectScope } from 'vue'
import type { ConnectionHost, ConversationContext } from '@numen/plugin-sdk'
import { WORDS } from '../words'
import { createAgentConnections, say, type TaskScript } from '../fixtures'
import { useConversationView } from './useConversationView'

const open = (
  context: Partial<ConversationContext> = {},
  script: TaskScript = { frames: [say('Because of the leaves.')] },
) => {
  const agent = createAgentConnections(script)
  const scope = effectScope()
  const view = scope.run(() =>
    useConversationView({ connections: agent.host, ...context }, (draw) => draw()),
  )!
  return { view, scope, calls: agent.calls }
}

describe('the conversation view', () => {
  it('runs the question on the connections it was given', async () => {
    const { view, calls } = open()

    await view.send('why is it called that')

    expect(calls.map((one) => [one.operation, one.request.prompt])).toStrictEqual([
      ['acp.task', 'why is it called that'],
    ])
  })

  // The card a question is about goes to the tools as their context, not into the question, where
  // a name would be read as an instruction.
  it('hands the tools the context and the vault it was given', async () => {
    const { view, calls } = open({
      toolContext: { 'md.numen.views.flashcards.studio/card': 'k3v9q2m8xa' },
      vault: 'vault-orchard',
    })

    await view.send('why')

    expect(calls[0]?.request.toolContext).toStrictEqual({
      'md.numen.views.flashcards.studio/card': 'k3v9q2m8xa',
    })
    expect(calls[0]?.request.vaultId).toBe('vault-orchard')
  })

  it('answers every question in one conversation', async () => {
    const { view, calls } = open()
    await view.send('one')
    await view.send('two')
    expect(calls[0]?.request.conversation).not.toBe('')
    expect(calls[0]?.request.conversation).toBe(calls[1]?.request.conversation)
  })

  it('never gives one name to two views', async () => {
    const first = open()
    const second = open()
    await first.view.send('why')
    await second.view.send('why')
    expect(first.calls[0]?.request.conversation).not.toBe(second.calls[0]?.request.conversation)
  })

  it('sends nothing when nothing was written', async () => {
    const { view, calls } = open()
    view.setQuestion('wh')

    await view.send('')

    expect(calls).toStrictEqual([])
    expect(view.question.value).toBe('wh')
  })

  it('clears what was written once it is sent', async () => {
    const { view } = open()
    view.setQuestion('why')

    await view.send('why')

    expect(view.question.value).toBe('')
    expect(view.turns.value.map((turn) => turn.text)).toContain('Because of the leaves.')
  })

  it('names the field as the caller says, and as the vault chat does where it says nothing', () => {
    expect(open({ placeholder: 'Ask about this card' }).view.placeholder).toBe(
      'Ask about this card',
    )
    expect(open().view.placeholder).toBe(WORDS.ask)
  })
})

describe('a question in flight', () => {
  /** Connections whose answer does not arrive until the test lets it. */
  const createSlowConnections = () => {
    let answer: () => void = () => undefined
    const hold = new Promise<void>((then) => {
      answer = then
    })
    const { host } = createAgentConnections({ frames: [], hold })
    return { host, answer: () => answer() }
  }

  const openView = (connections: ConnectionHost) => {
    const scope = effectScope()
    return {
      scope,
      view: scope.run(() => useConversationView({ connections }, (draw) => draw()))!,
    }
  }

  it('is working while an answer is on its way, and not before or after', async () => {
    const slow = createSlowConnections()
    const { view } = openView(slow.host)
    expect(view.isAnswerPending.value).toBe(false)

    const reply = view.send('why')
    expect(view.isAnswerPending.value).toBe(true)

    slow.answer()
    await reply
    expect(view.isAnswerPending.value).toBe(false)
  })

  it('stops working when the answer on its way is stopped', async () => {
    const slow = createSlowConnections()
    const { view } = openView(slow.host)
    const reply = view.send('why')

    view.stop()

    expect(view.isAnswerPending.value).toBe(false)
    await reply
  })

  it('stops the answer on its way when the view is taken away', async () => {
    const slow = createSlowConnections()
    const { view, scope } = openView(slow.host)
    const reply = view.send('why')

    scope.stop()

    expect(view.isAnswerPending.value).toBe(false)
    await reply
  })
})
