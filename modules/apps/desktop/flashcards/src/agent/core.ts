/**
 * Asking about the card in front of a person.
 *
 * Nothing here draws: the port is `AgentPort`, `@numen/wire` says what the
 * steps mean, and this says what the question is about.
 */
import { createClient } from '@connectrpc/connect'
import { AgentService } from '@numen/protocol'
import { agentPort } from '@numen/wire'

import { transport } from '../transport'
import type { Asked } from '../core'

export const core = agentPort(createClient(AgentService, transport))

/**
 * The card a question is about, written where the agent will read it.
 *
 * The deck is the file in front of the person and goes as the focus; the card
 * is a mark inside that file, and the tools address it by that mark.
 */
export const standing = (card: Asked): string =>
  `The card is \`${card.card}\` in \`${card.deck}\`, shown through its ${card.face} face. ` +
  `Read it with card_read before answering.`
