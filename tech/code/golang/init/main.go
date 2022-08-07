package main

import (
	"fmt"
	_ "init/b"
)

func init() {
	fmt.Println("main,", hello)
}

func init() {
	fmt.Println("main,", hello2)
}

func main() {

}

var hello = "hello"
var hello2 = "hello2"
