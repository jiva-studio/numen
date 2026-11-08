import type { AgentHost, AgentQuery, AgentStep } from '../agent'

// MockAgentHost is an AgentHost that returns the same steps for every query and
// records the queries, so a test can read the queries a plugin sent.
export interface MockAgentHost extends AgentHost {
  // Every query received, in order.
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
      for await (const step of steps) yield step
    },
    finish: (conversation) => {
      finished.push(conversation)
      return Promise.resolve()
    },
    resolveLinks: () => Promise.resolve([]),
    getUnreachable: () => unreachable,
    getSetting: () => undefined,
    // The mock host keeps no settings.
    writeSetting: () => Promise.resolve(),
    getModelsAt: () => [],
    getQueries: () => [...queries],
    getFinished: () => [...finished],
    setUnreachable: (why) => {
      unreachable = why
    },
  }
}
