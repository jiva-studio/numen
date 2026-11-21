/**
 * The conversation another plugin mounts by the `conversation` capability: one conversation on the
 * connections it passed, answered by the agent and the model the person chooses among them, with the tool context and the vault it passed, stopped when the view is
 * taken away.
 */
import { onScopeDispose, ref } from 'vue'
import type { Paint } from '@numen/ui'
import type { ConversationContext } from '@numen/plugin-sdk'
import { useConversation } from '../thread'
import { createAgentList, useAgentChoice } from './agents'
import { WORDS } from '../words'

export function useConversationView(context: ConversationContext, paint?: Paint) {
  const list = createAgentList(context.connections, WORDS)
  const agents = useAgentChoice(context.connections, list)
  const conversation = useConversation({
    connections: context.connections,
    getPreferredRow: agents.getPreferredRow,
    words: WORDS,
    conversation: `conversation:${crypto.randomUUID()}`,
    toolContext: context.toolContext,
    vault: context.vault,
    paint,
  })
  const question = ref('')

  const reloadAgents = () => {
    list.reload().catch(() => undefined)
  }
  const subscription = context.connections.onChange?.(reloadAgents)
  reloadAgents()

  const setQuestion = (text: string) => {
    question.value = text
  }

  const send = async (text: string) => {
    if (!text) return
    question.value = ''
    await conversation.ask(text)
  }

  onScopeDispose(() => {
    subscription?.dispose()
    conversation.stop()
  })

  return {
    agents,
    question,
    turns: conversation.turns,
    isAnswerPending: conversation.isAnswerPending,
    placeholder: context.placeholder ?? WORDS.ask,
    setQuestion,
    send,
    stop: conversation.stop,
  }
}
