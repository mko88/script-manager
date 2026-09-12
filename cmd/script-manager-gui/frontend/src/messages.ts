import messages from '../../../../internal/messages/gui.json'
import { createMessages, type FlattenKeys } from '@shared/messages'

export type MessageKey = FlattenKeys<typeof messages>

export const { t, setMessageOverride } = createMessages(messages)
