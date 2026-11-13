export { createHost } from './host'
export type { HostOptions } from './host'
export {
  callTool,
  createAgentConnections,
  endTool,
  failWith,
  say,
  showPlan,
  stopFor,
  think,
  writeTool,
} from './connections'
export type { AgentListing, FrameInit, TaskScript } from './connections'
export { byWholeText } from './text'
