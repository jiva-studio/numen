/**
 * The conversation another plugin mounts by the `conversation` capability: one conversation on the
 * connections it passed, with the tool context and the vault it passed, stopped when the view is
 * taken away.
 */
import { onScopeDispose, ref } from 'vue'
import type { Paint } from '@numen/ui'
import type { ConversationContext } from '@numen/plugin-sdk'
import { useConversation } from '../thread'
import { WORDS } from '../words'

export function useConversationView(context: ConversationContext, paint?: Paint) {
  const conversation = useConversation({
    connections: context.connections,
    words: WORDS,
    conversation: `conversation:${crypto.randomUUID()}`,
    toolContext: context.toolContext,
    vault: context.vault,
    paint,
  })
  const question = ref('')

  const setQuestion = (text: string) => {
    question.value = text
  }

  const send = async (text: string) => {
    if (!text) return
    question.value = ''
    await conversation.ask(text)
  }

  onScopeDispose(() => conversation.stop())

  return {
    question,
    turns: conversation.turns,
    isAnswerPending: conversation.isAnswerPending,
    placeholder: context.placeholder ?? WORDS.ask,
    setQuestion,
    send,
    stop: conversation.stop,
  }
}
