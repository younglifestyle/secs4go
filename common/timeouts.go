package common

type Timeouts struct {
	T3 int
	T5 int
	T6 int
}

func NewTimeouts() *Timeouts {
	return &Timeouts{
		T3: 45,
		T5: 10,
		T6: 5,
	}
}
