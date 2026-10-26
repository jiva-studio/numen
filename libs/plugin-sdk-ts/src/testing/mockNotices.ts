import type { NoticeHost, NoticeTask, NoticeTone } from '../notices'

// MockNoticeHost is a NoticeHost that holds what it is given, so a test can read what a plugin said.
export interface MockNoticeHost extends NoticeHost {
  // Every notice shown, in the order it was, with the tone it was shown in.
  getShown(): readonly { readonly text: string; readonly tone: NoticeTone }[]
  // The list of work the plugin gave last.
  getTasks(): readonly NoticeTask[]
}

// createMockNoticeHost builds a notice host that keeps what it is given in memory.
export function createMockNoticeHost(): MockNoticeHost {
  const messages: { text: string; tone: NoticeTone }[] = []
  let tasks: readonly NoticeTask[] = []
  return {
    show: (text, tone = 'plain') => {
      messages.push({ text, tone })
    },
    setTasks: (next) => {
      tasks = next
    },
    getShown: () => [...messages],
    getTasks: () => tasks,
  }
}
