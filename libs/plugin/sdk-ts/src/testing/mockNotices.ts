import type { NoticeHost, NoticeTask, Severity } from '../notices'

// MockNoticeHost is a NoticeHost that records what it receives, so a test can read what a plugin reported.
export interface MockNoticeHost extends NoticeHost {
  // Every notice shown, in the order it was, with the severity it was shown in.
  getShown(): readonly { readonly text: string; readonly severity: Severity }[]
  // The list of work the plugin gave last.
  getTasks(): readonly NoticeTask[]
}

// createMockNoticeHost builds a notice host that keeps what it is given in memory.
export function createMockNoticeHost(): MockNoticeHost {
  const messages: { text: string; severity: Severity }[] = []
  let tasks: readonly NoticeTask[] = []
  return {
    show: (text, severity = 'info') => {
      messages.push({ text, severity })
    },
    setTasks: (next) => {
      tasks = next
    },
    getShown: () => [...messages],
    getTasks: () => tasks,
  }
}
