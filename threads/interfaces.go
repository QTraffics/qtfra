package threads

type Safe interface {
	ThreadSafe() bool
}
