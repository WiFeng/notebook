package main

import "fmt"

type ByteSlice []byte

func (p *ByteSlice) Write(data []byte) (n int, err error) {
	slice := *p
	slice = append(slice, data...)
	*p = slice
	return len(data), nil
}

func main() {
	var b ByteSlice
	// fmt.Fprintf(&b, "This hour has %d days\n", 7)

	b.Write([]byte("hello world"))
	fmt.Printf("b:%v\n", string(b))

}
