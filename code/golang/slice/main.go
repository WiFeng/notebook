package main

import "fmt"

func main() {
	s := make([]int, 0, 5)
	s1 := s
	fmt.Println(len(s), len(s1))

	s = append(s, 1, 2, 3)
	fmt.Println(len(s), len(s1))

	m := make(map[int]bool, 2)
	m1 := m
	fmt.Println(len(m), len(m1))

	m[1] = true
	m[2] = false
	m[3] = false
	m[4] = false
	m[5] = false
	m[6] = false
	m[7] = false
	m[8] = false

	fmt.Println(len(m), len(m1))
}
