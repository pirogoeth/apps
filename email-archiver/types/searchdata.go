package types

import "github.com/emersion/go-imap/v2"

type (
	SearchDataSender   chan<- *SearchData
	SearchDataReceiver <-chan *SearchData
)

// SearchData contains the payload that will be ingested into
// the indexer
type SearchData struct {
	Mailbox  *MailboxData
	Envelope *imap.Envelope
}

type MailboxData struct {
	Mailhost string `json:"mailhost"`
	Username string `json:"username"`
	Mailbox  string `json:"mailbox"`
}
