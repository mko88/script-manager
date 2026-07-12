// The JSON lives in internal/messages so the Go backend can embed it directly
// too — this import is the single canonical copy, not a duplicate. The t()/
// override machinery is shared with sm-config-edit via @shared/messages.
import messages from '../../../../internal/messages/gui.json'
import { createMessages, type FlattenKeys } from '@shared/messages'

export type MessageKey = FlattenKeys<typeof messages>

export const { t, setMessageOverride } = createMessages(messages)
