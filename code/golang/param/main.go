package main

type S struct {
	x int
	y int
}

func f(s S) {
	s.x = 1
}

func f2(s *S) {
	s.y = 2
}

func main() {

	// s := S{}
	// ss := &S{}
	// // f(s)
	// // f2(s)

	// f(ss)
	// f2(ss)
}
