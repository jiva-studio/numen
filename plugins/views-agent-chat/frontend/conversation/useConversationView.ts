/**
 * The conversation another plugin mounts by the `conversation` capability: one conversation with
 * the port it passed, about the focus it passed, finished when the view is taken away.
 */
import { onScopeDispose, ref } from 'vue'
import type { Paint } from '@numen/ui'
import type { ConversationContext } from '@numen/plugin-sdk'
import { useConversation } from '../thread'
import { WORDS } from '../words'

export function useConversationView(context: ConversationContext, paint?: Paint) {
  const conversation = useConversation({
    agent: context.agent,
    words: WORDS,
    conversation: `conversation:${crypto.randomUUID()}`,
    paint,
  })
  const question = ref('')

  const setQuestion = (text: string) => {
    question.value = text
  }

  /** What the person wrote is sent alone; what it is about goes as the focus beside it. */
  const send = async (text: string) => {
    if (!text) return
    question.value = ''
    await conversation.ask(text, context.focus)
  }

  /** What stands where the answers do while there are none. */
  const getSilence = () => context.getUnreachable?.() || WORDS.nothingSaid

  onScopeDispose(() => conversation.finish())

  return {
    question,
    turns: conversation.turns,
    isWorking: conversation.isWorking,
    placeholder: context.placeholder ?? WORDS.ask,
    setQuestion,
    send,
    stop: conversation.stop,
    getSilence,
  }
}

export type ConversationViewState = ReturnType<typeof useConversationView>
