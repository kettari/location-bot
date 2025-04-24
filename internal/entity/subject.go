package entity

type SubjectType string

const (
	SubjectTypeNew            SubjectType = "new"
	SubjectTypeBecomeJoinable SubjectType = "become_joinable"
	SubjectTypeCancelled      SubjectType = "cancelled"
)

type subject interface {
	Register(observer Observer)
	notifyAll(subject SubjectType)
}
