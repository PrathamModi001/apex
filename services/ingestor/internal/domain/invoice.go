package domain

import "time"

type Source string

const (
	SourceGmail    Source = "gmail"
	SourceTelegram Source = "telegram"
	SourceTest     Source = "test"
)

type RawInvoice struct {
	ID         string
	Source     Source
	FileKey    string
	SHA256     string
	Sender     string
	ReceivedAt time.Time
	Metadata   map[string]string
}
