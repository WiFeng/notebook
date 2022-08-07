# Golang 编程进阶之（二）

## init

* init 的执行是在当前包中的所有变量、所有常量之后。变量可以定义在init之后，或者是独立的文件中。与其在文件中的定义前后顺序无关，是否在同一个文件中也无关。
* 一个文件中可以定义多个init，其执行顺序是其定义的先后顺序。
* 一个包中的多个文件中都可以定义init，当其他包导入这个包时，其init顺序先按照文件名按照字典序执行，在同文件中按照上一条规则执行。
* 当一个程序 a 导入一个包 b 时，会按照这样的顺序执行：b 变量初始化 -> b init 执行 -> a 变量初始化 -> a init 执行。当 b 中 有导入包 c 时，按照同样的方式递归执行。
* 在 sql / pprof 包中正式利用 init 的这种机制实现了功能的引入，只需要 import 导入即可，无需额外的初始化代码。

## defer

* defer 常常用在一些需要执行收尾操作的地方。如 io 操作，锁操作。defer 在部分web框架中也用来捕获请求执行时间，执行状态。
* 为了 defer 可以有效的执行，我们一般会把 defer 代码放在最靠前的位置，防止一些程序逻辑导致无法执行defer操作。
* defer 调用应用在函数中，在 return 语句之后执行。也就是对于普通类型变量，defer 中对变量的修改不会影响函数的实际返回结果。当函数的返回值用变量名的方式声明时，那么在defer中这个变量的修改会改变函数实际的返回结果的。
* defer 的执行并不是异步的，无论上面2种情况中的哪种，defer调用执行完毕后，调用方才会接收到返回值。因此在defer中执行操作也需要考虑其响应时间。
* 在同一函数中可以执行多个defer 调用，其中执行顺序与定义顺序相反。
* recovery 操作一般在 defer 中进行。据我所知，也只能在 defer 中调用，否则将无法进行有效捕获。也就是说，当发生了异常时，defer函数依然会执行。意识到这一点，则我们可以在这个阶段进行一些高级操作。

```golang
func d1() {
    fmt.Println("d1")
}

func d2() {
    fmt.Println("d2")
}

func d() int {
    var i = 2

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
```

```plain
// OUTPUT

d,i: 4
d2
d1
main,r: 3
```

```golang
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
    return
}

func main() {
    r := d()
    fmt.Println("main,r:", r)
}
```

```plain
// OUTPUT

d,i: 4
d2
d1
main,r: 4
```

## select

* `select`操作用于在接收操作、发送操作集合中选择可以执行的操作执行。与 switch 很像，但是 select 的 case 都是通讯的操作。 
* select 可以定义最多一个 default case，可以出现在 case 列表的任意位置。switch中的default 只能定义在 case 列表的最下方，否则在default case 之后的 case 则永远无法执行。


## switch type



## new 与 make 区别

## 类方法定义

## 参数传递


## 错误模式


## 内存模型
