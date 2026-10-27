/**
 * The notices a window is handed by the parts it hosts, each under the name of
 * the part that gave it, as one list for the corner to draw.
 */
import { computed, shallowRef, type ComputedRef } from 'vue'
import { createNotice, type Notice, type Task } from '../lib/notice'
import type { Severity } from '../lib/tally'

/** The notices of every source, and what changes them. */
export interface NoticeQueue {
  /** The tasks of every source first, then the messages, oldest first. */
  readonly notices: ComputedRef<readonly Notice[]>
  /** A notice from a source. An info notice goes once read; any other stands until dismissed. */
  show(source: string, text: string, severity: Severity): void
  /** The work a source is doing, replacing the list it gave last. */
  setTasks(source: string, tasks: readonly Task[]): void
  /** A notice the person is finished with, by its identity. */
  dismiss(id: string): void
  /** Everything a source gave, gone with the source. */
  clear(source: string): void
}

interface Message {
  readonly source: string
  readonly notice: Notice
}

export function createNoticeQueue(): NoticeQueue {
  const messages = shallowRef<readonly Message[]>([])
  const work = shallowRef<ReadonlyMap<string, readonly Notice[]>>(new Map())
  let count = 0

  const notices = computed<readonly Notice[]>(() => [
    ...[...work.value.values()].flat(),
    ...messages.value.map((one) => one.notice),
  ])

  const show = (source: string, text: string, severity: Severity): void => {
    if (text === '') return
    count += 1
    const notice: Notice = {
      id: `${source}#${count}`,
      text,
      severity,
      stay: severity === 'info' ? 'read' : 'kept',
      isWorking: false,
      isUserStarted: true,
    }
    messages.value = [...messages.value, { source, notice }]
  }

  const setTasks = (source: string, tasks: readonly Task[]): void => {
    const next = new Map(work.value)
    if (tasks.length === 0) next.delete(source)
    else
      next.set(
        source,
        tasks.map((task) => createNotice({ ...task, id: `${source}:${task.id}` })),
      )
    work.value = next
  }

  const dismiss = (id: string): void => {
    messages.value = messages.value.filter((one) => one.notice.id !== id)
  }

  const clear = (source: string): void => {
    messages.value = messages.value.filter((one) => one.source !== source)
    setTasks(source, [])
  }

  return { notices, show, setTasks, dismiss, clear }
}
