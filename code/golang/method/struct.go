package main

import "fmt"

type S struct {
	x    int
	sum  int
	sum2 int
}

func (s S) add(y int) {
	s.sum = s.x + y
}

func (s *S) add2(y int) {
	s.sum2 = s.x + y
}

func main() {
	s := S{
		x: 1,
	}
	s.add(2)
	s.add2(2)
	fmt.Println(s.sum, s.sum2)

	s1 := &S{
		x: 1,
	}
	s1.add(2)
	s1.add2(2)
	fmt.Println(s1.sum, s1.sum2)
}
