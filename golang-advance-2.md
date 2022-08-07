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

* `select`操作用于在接收操作、发送操作集合中选择可以执行的操作执行。与 `switch` 很像，但是 `select` 的 case 都是通讯的操作。
* 在 `select` 中最多只能定义一个 `default case`，其可以出现在 case 列表的任意位置。`switch`中的 default 只能定义在 case 列表的最下方，否则在 `default case` 之后的 case 则永远无法执行。
* select 执行分位以下几个步骤：
  * 在进入`select`前，所有的case都会进行评估，按照定义顺序、`exactly once`(仅且只有一次)，其中包括 recv 声明中的 channel, send 声明中的channel 以及其右侧的表达式。无论是否有 case 会选中执行，在这个评估过程中的副作用总会产生。参考下方的示例可知，虽然实际执行了 `default case`，但是 f 函数依然被调用了。
  * 如果一个或者多个通信可以执行（就是除default之外的其他case），会随机选择一个执行。否则，如果有 `default case` ， 则执行`default case`，如果没有 `default case`，这个 `select` 将一致阻塞直到至少有一个通信（communication）可以执行。
  * 除非选中的是 `default case`，否则通信操作将被执行。
  * 如果选中的 case 是 recv 声明（短变量声明或者赋值），左侧的表达式将被评估，并且把接收到的值进行赋值。
  * 选中的 case 被执行。
* 在 `select` 执行中，有提到评估、执行2种说法，评估其实就是执行，只是把中间步骤中执行统一使用评估一词来描述，为了直观上的执行进行区分。
* 因为在一个 `nil` 的 channel 上通信永远不会发生，因此只有 `nil` channel 且没有 `default case` 的 `select` 将永远阻塞。

```golang
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
        //    a[f()] = t
    default:
        print("no communication\n")
    }

    _ = c
}
```

```plain
// OUTPUT

f()
no communication
```

```golang
for {  // send random sequence of bits to c
    select {
    case c <- 0:  // note: no statement, no fallthrough, no folding of cases
    case c <- 1:
    }
}

select {}  // block forever
```

## Type switches

* type switch 比较类型而不是比较值。除此之外，它跟 expression switch 很像。
* 除了普通类型可以作为 case 之外，预定义的 nil 也可以。当这个switch 求解的变量是一个nil类型时，这个case被选中执行。
* 只有 interface 类型可以执行 switch type 操作。其他明确的类型是不需要的，也是不允许的。
* switch 类型断言的使用切勿进行完全封装，把所有类型都接收回来进行一次处理。golang 提供了这种方法是需要活学活用，切勿把一切类型都通过统一的一个函数封装来实现，因为在不同的场景，允许的类型集合可能不是不同的，对于那些上下文中不支持的类型我们要给出错误提示。

```golang
switch i := x.(type) {
case nil:
    printString("x is nil")                // type of i is type of x (interface{})
case int:
    printInt(i)                            // type of i is int
case float64:
    printFloat64(i)                        // type of i is float64
case func(int) float64:
    printFunction(i)                       // type of i is func(int) float64
case bool, string:
    printString("type is bool or string")  // type of i is type of x (interface{})
default:
    printString("don't know the type")     // type of i is type of x (interface{})
}
```

## new 与 make 区别

* golang 中有2个分配内存的原语（primitives），分别是 `new` 与 `make`。它们做不同的事情，应用于不同的数据类型。很容易混淆，但是其实规则很简单。
* 在 golang 中，`new` 返回指定类型新申请的内存，并且用零值填充。简言之，新创建了这样这个类型数据，并且值全部初始化为结构体中各类型的默认值，并返回其指针（`*T`）。可以很安全的使用，不会因为空指针异常而烦恼。
* make 只应用于创建 `slice / map / channel`，并且返回一个初始化的非零值类型（是`T`，并不是 `*T`）。
* 通过 new 创建 slice 时，并不会初始化slice内部的实际存储区域，也就是说通过new 创建的slice 其指向的存储区域为nil, 因此并不能直接使用。总之，在大多数场景下，`slice / map / channel` 这些操作上 `new` 根本不需要使用，只能使用 `make`就好。
* `make` 返回的是值类型，并不是指针类型；如果要显式的获取其指针类型，可以通过`new`来创建进而明确的得到其变量的地址。

## 方法定义

* 对于任何指定的类型都可以定义方法（除指针 与 `interface`类型），并不是必须要是一个结构体。
* 指针与值的区别。指针很明确在方法内部的修改会被调用方可见。但是指针是否生效是方法的定义决定的，与方法调用时变量是否是指针类型无关。如果定义的方法是指针类型，即使调用时变量使用的是值类型，只要这个变量地址是可获取的，那么编译器会自动转换为 `&s.add` 方式调用。如果定义的方式是值类型，即使调用时变量使用是的指针类型，在编译阶段也会预处理，实际接收到是原有变量值的副本，因此原有变量并不会被修改。
* 标准是：值类型的方法允许接收值类型的变量与指针类型的变量，但是指针类型的方法只允许接收指针类型的变量。在这个标准之上，编译器为我们自动进行了一些处理，最终就是上一条的规则。

```golang
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
```

```plain
// OUTPUT

0 3
0 3
```

## 参数传递

* 普通函数与类中方法的参数遵循相同的规则。
* 基础类型按照值传递，指针类型则按照地址传递。通过 `make` 创建的 `slice / map / chan` 这些类型是值类型，但是其结构内部包含数据区域的指针，因此实际传递效果与指针类型一致。interface 类型则与动态的值类型有关。
* 函数参数、类中方法的参数，定义的是指针类型，则调用时也需要指针类型；定义时是值类型，则调用时也需要是值类型。如果类型不匹配，则出现编译错误。这里编译器并不会进行任何自动转换操作。

## 错误模式

* 错误类型，共可以归为四大类
  * 编译错误。如果没有按照语言规范的写法，用法，类型错误，参数错误，将在编译阶段给出错误信息。
  * 运行时 panic 错误。在 golang中，如果存在 `data race` / 数组越界 / map 未初始化就赋值 等等操作均会给出 panic 错误，结果就是程序直接终止运行。任何 goroutine 中的 panic 错误，均会使整个主进程停止运行。
    * 可以通过在 `defer` 中调用 `recovery` 捕获错误来阻止影响到主程序的运行。
    * 可以我们主动调用 `panic` 方法来触发一个运行时错误。
  * 运行时逻辑错误。这是函数返回的 error 类型错误，这个在业务代码中会很频繁的使用到。
  * 运行是bool类型的 ok 标识符。这种也可以认为是错误，也可以理解为一种技巧。但它释放了一种信号，表示其与普通状态不同。
* golang 中并没有 Warn 错误, Notice 错误。因为它是编译语言，强类型语言，在编译阶段已经对一些可以可以检查到的错误进行提示，这样为工程师的粗心大意提供了很好的弥补。
* 当有 data race 时，通过在运行时添加额外参数，可以产生数据竞争的执行代码输入，这样可以方便定位问题。不过，这也属于一种debug模式。

## 内存模型

* Go 的内存模型指定了这样的条件，在其中一个协程中对一个变量的读操作可以保证其观察到在另一个协程中同一个变量的写入操作。要了解详细内容请参考官方文档：https://golang.google.cn/ref/mem
* 多个协程同时访问、修改相同的数据，那么必须序列化这些访问，即顺序访问。为了实现序列化访问，可以通过 channel 操作或者其他同步原语（如sync包中的方法与 sync/atomic包含的方法）来保护数据。
* data race(数据竞争)是指对一个内存位置写入时，其他协程在相同的位置也正在执行读或者写操作，除非所有牵涉的数据访问都是原子访问（由sync/atomic包提供的）。我们应该尽可能通过合适的同步方式避免 data race 发生。
