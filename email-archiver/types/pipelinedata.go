package types

import "github.com/emersion/go-imap/v2"

// ArchiverPipelineData is a conglomerate struct that will contain
// all data to flow through the pipeline
type ArchiverPipelineData struct {
	Mailbox  *MailboxData
	Envelope *imap.Envelope
	Body     []byte
	// UID is a composite ID of (Mailbox name, Mailbox UIDVALIDITY, Message UID) to assign a completely unique identifier to each message
	UID ImapUID
}

type MailboxData struct {
	Mailhost  string `json:"mailhost"`
	Username  string `json:"username"`
	Mailbox   string `json:"mailbox"`
	InboxAddr string `json:"inbox_addr"`
}
