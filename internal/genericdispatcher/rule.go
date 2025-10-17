package genericdispatcher

type Rule[R any] struct {
	Test    func(R) bool
	Execute func(R) error
}
