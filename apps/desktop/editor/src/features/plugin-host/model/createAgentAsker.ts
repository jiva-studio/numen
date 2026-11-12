import { formatError, type Caught } from '@/shared/api/errors'
import type { WindowTabs } from '@/entities/tab'
import type { MessageWriter } from '@/shared/model/notices/messages'

/** Puts a question to the tab of a kind that takes text, or says that none is open. */
export function createAgentAsker(
  windowTabs: Pick<WindowTabs, 'groups'>,
  writeMessage: MessageWriter,
): (text: string) => void {
  return (text) => {
    const textKind = windowTabs.groups.findKind((kind) => kind.events?.receiveText !== undefined)
    if (!textKind) {
      writeMessage('The agent chat is not available: its plugin is off or did not start.', 'error')
      return
    }
    windowTabs.groups
      .sendText(textKind.kind, text)
      .catch((error: Caught) => writeMessage(formatError(error), 'error'))
  }
}
