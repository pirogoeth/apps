package types

import (
	"fmt"

	"github.com/emersion/go-imap/v2"
)

type ImapUID struct {
	mailbox     string
	uidValidity uint32
	messageUID  imap.UID
}

func NewImapUID(mailbox string, uidValidity uint32, messageUID imap.UID) ImapUID {
	return ImapUID{
		mailbox:     mailbox,
		uidValidity: uidValidity,
		messageUID:  messageUID,
	}
}

// String returns the internal representation of the ImapUID
func (uid ImapUID) String() string {
	return fmt.Sprintf("%s-%d-%d", uid.mailbox, uid.uidValidity, uid.messageUID)
}

// MailboxName returns the name of the mailbox
func (uid ImapUID) MailboxName() string {
	return uid.mailbox
}

// MailboxUIDValidity returns the UIDVALIDITY of the mailbox
func (uid ImapUID) MailboxUIDValidity() uint32 {
	return uid.uidValidity
}

// MessageUID returns the ID of the message _within_ the mailbox context.
func (uid ImapUID) MessageUID() imap.UID {
	return uid.messageUID
}
