package console

type Command interface {
	Name() string
	Description() string
	Run() error
}
