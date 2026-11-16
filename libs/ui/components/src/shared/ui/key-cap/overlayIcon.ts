/**
 * The key next to the space bar on the keyboard a browser reports it is running
 * on: Command on Apple keyboards, Control everywhere else.
 */
export const overlayIcon = (agent: string): 'command' | 'control' =>
  /mac|iphone|ipad|ipod/i.test(agent) ? 'command' : 'control'
