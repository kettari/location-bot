package entity

type MessageDispatcher interface {
	Send([]string) error
}
