package main

import "fmt"

func f() int {
	fmt.Println("f()")
	return 2
}

func main() {
	var a []int
	var c, c1, c2, c3, c4 chan int
	var i1, i2 int
	// c2 = make(chan int, 2)

	select {
	case i1 = <-c1:
		print("received ", i1, " from c1\n")
	case c2 <- f():
		print("sent ", i2, " to c2\n")
	case i3, ok := (<-c3): // same as: i3, ok := <-c3
		if ok {
			print("received ", i3, " from c3\n")
		} else {
			print("c3 is closed\n")
		}
	case a[f()] = <-c4:
		// same as:
		// case t := <-c4
		//	a[f()] = t
	default:
		print("no communication\n")
	}

	_ = c
}
