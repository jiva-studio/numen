/** What an agent tab displays: the composer, the answers, and the tab's own name. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  ask: 'Ask about the vault',
  send: 'Send',
  stop: 'Stop the agent',
  thinking: 'Thinking',
  unreachable: 'Agent unreachable.',
  nothing: 'The agent returned no answer.',
  maxTokens: 'The answer reached the length limit of the model.',
  maxTurnRequests: 'The agent reached its limit of requests for one question.',
  refusal: 'The agent refused to answer.',
  unsent: 'Not sent',
  stopped: 'Stopped',
  noTranscript: 'No messages yet',
  agent: 'Agent',
  reasoning: 'Reasoning',
  /** The model of an agent whose row names none, which the agent chooses itself. */
  defaultModel: 'Default model',
  /** The model of an agent whose row names none where its type requires one. */
  chooseModel: 'Choose a model',
  /** The model control where no agent can answer. */
  noAgent: 'No agent',
  noAgentMessage: 'No agent is set up.',
  openSettings: 'Open settings',
  /** The row of an agent in the model list while its models are being asked. */
  loading: 'Loading…',
}
