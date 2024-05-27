package inf

type Task interface {
	Run() error
	Close() error
}
