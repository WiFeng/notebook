package main

import (
	"fmt"
)

func d1() {
	fmt.Println("d1")

}

func d2() {
	fmt.Println("d2")
}

func d() (i int) {
	i = 2

	defer d1()
	defer d2()
	defer func() {
		i++
		fmt.Println("d,i:", i)
	}()

	i = 3
	return i
}

func main() {
	r := d()
	fmt.Println("main,r:", r)
}
