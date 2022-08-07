package b

import (
	"fmt"
	_ "init/a"
)

var hello = "hello"

func init() {
	fmt.Println("init/b/yy.go,", hello)
}
