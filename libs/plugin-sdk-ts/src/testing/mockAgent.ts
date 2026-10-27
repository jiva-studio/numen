import type { AgentHost, AgentQuery, AgentStep } from '../agent'

// MockAgentHost is an AgentHost that answers every question with the same steps and keeps what it
// was asked, so a test can read the questions a plugin put.
export interface MockAgentHost extends AgentHost {
  // Every question asked, in the order it was.
  getQueries(): readonly AgentQuery[]
  // Every conversation finished, in the order it was.
  getFinished(): readonly string[]
  // Sets why the agent cannot be reached; empty is reachable.
  setUnreachable(why: string): void
}

// createMockAgentHost builds an agent host that answers each question with the steps given.
export function createMockAgentHost(steps: readonly AgentStep[] = []): MockAgentHost {
  const queries: AgentQuery[] = []
  const finished: string[] = []
  let unreachable = ''
  return {
    async *ask(query) {
      queries.push(query)
      yield* steps
    },
    finish: async (conversation) => {
      finished.push(conversation)
    },
    resolveLinks: async () => new Map(),
    getUnreachable: () => unreachable,
    getSetting: () => undefined,
    writeSetting: async () => {},
    getModelsAt: () => [],
    getQueries: () => [...queries],
    getFinished: () => [...finished],
    setUnreachable: (why) => {
      unreachable = why
    },
  }
}
