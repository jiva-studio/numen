import { computed, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { LinkRole } from '@/entities/identity'
import { getLinkView } from '../lib/phases'
import type { useLinkSession } from './useLinkSession'

/** What the person typed into the dialog and which way it links. */
export interface LinkDraft {
  text: Ref<string>
}

const ignore = (): void => undefined

/**
 * The view a link dialog is in and what its buttons do there. A joiner starts at the scan of the
 * link; a host shows its own QR code at once.
 */
export function useLinkActions(
  session: ReturnType<typeof useLinkSession>,
  role: LinkRole,
  draft: LinkDraft,
) {
  const view = computed(() =>
    getLinkView({
      link: session.link.value,
      role,
      isScanning: role === 'joiner',
      isStalled: session.isStalled.value,
      hasProblem: session.problem.value !== '',
    }),
  )
  const run = (work: Promise<void>): void => runInBackground(work, ignore)

  function submit(): void {
    switch (view.value) {
      case 'scan':
        run(session.start('scan', draft.text.value))
        break
      case 'peer':
        run(session.answers.decide(true))
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

  return { view, submit, decline, scan }
}
