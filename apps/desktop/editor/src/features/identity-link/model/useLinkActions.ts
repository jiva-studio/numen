import type { Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { LinkView } from '../lib/phases'
import type { useLinkSession } from './useLinkSession'

/** What the person typed into the dialog and which way it links. */
export interface LinkDraft {
  text: Ref<string>
  code: Ref<string>
}

const ignore = (): void => undefined

/** What the buttons of the link dialog do in the view it is in. */
export function useLinkActions(
  session: ReturnType<typeof useLinkSession>,
  draft: LinkDraft,
  view: Readonly<Ref<LinkView>>,
) {
  const run = (work: Promise<void>): void => runInBackground(work, ignore)

  function submit(): void {
    switch (view.value) {
      case 'scan':
        run(session.start('scan', draft.text.value))
        break
      case 'peer':
        run(session.answers.decide(true))
        break
      case 'code':
        run(session.answers.sendCode(draft.code.value))
        break
      case 'account':
        run(session.answers.confirmJoin(true))
        break
      case 'stalled':
      case 'refused':
      case 'failed':
        run(session.retry())
        break
      default:
        break
    }
  }

  function decline(): void {
    if (view.value === 'peer') run(session.answers.decide(false))
    if (view.value === 'account') run(session.answers.confirmJoin(false))
  }

  function scan(text: string): void {
    run(session.start('scan', text))
  }

  return { submit, decline, scan }
}
