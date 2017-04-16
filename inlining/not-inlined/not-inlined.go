package main

func dummy() {}

func ClosureCreation() {
	_ = func() {}
}

var x = []byte{0}

func RangeLoop() {
	for _ = range x {
	}
}

var ch chan struct{}

func Select() {
	select {
	case <-ch:
	default:
	}
}

func Go() {
	go dummy()
}

func Defer() {
	defer dummy()
}

func LocalTypeDecl() {
	type I int
}

func main() {
	ClosureCreation()
	RangeLoop()
	Select()
	Go()
	Defer()
	LocalTypeDecl()
}
