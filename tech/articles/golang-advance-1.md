# Golang 编程进阶之（一）

Golang 是业界公认的最有潜力的语言之一，其已经被大量应用在云原生、容器化编排、微服务、区块链应用、web应用等等。Google SRE 团队在内部中间件支持的3种编程语言分别为 C++ / java / golang，也进一步验证了其应用的广泛程度。

Golang 语言是一个比较简单的语言，其设计理念中包括“易用性”、“容易理解”、“高性能”、“高并发”这些理念，这与 Redis 设计理念如出一辙。Redis 同样也成为了每个技术人员必须使用的核心组件，我想这些核心思想可以使这些语言、组件、工具走的更远，同时也是我们这些晚辈需要刻意学习的地方。

使用 Golang 开始构建一个web服务非常容易，包括使用一些高级功能也并不是非常难。我们作为每一个普通的大脑都可以学会，并不必担忧。但是有些技巧使用频率低，为了更好的进行记忆，我想进行一些总结。整个系列包括三篇，第一篇介绍 channel / goroutine / map / slice 的一些技巧，第二篇介绍 类、参数、内存模型、错误模式，第三篇介绍垃圾回收机制，第四篇减少协程调度。这些知识点看似简单，但要铭记与心其实是比较难的，这点我有非常深的感触。在平常工作中，我们要熟悉业务，结合业务设计技术架构、技术方案，要学习一些前言技术、也要阅读一些有助于个人成功的非技术类书籍，我们并不能够把每天的时间放在一个点上。但是不能聚焦并不是一件好事情，导致的结果是很难在某个领域成为专家，这对技术人员来说会导致失去竞争力，适当把握度，每天提醒自己“聚焦”。

## map

* map 使用前需要先初始化，否则其为nil, 执行赋值、取值操作会出现空指针错误。
* 在初始化后，执行取值操作，针对不存在的key，并不会出现空指针错误。但是有时程序遇到这种情况却给出panic错误，是因为map中的存放的val是指针类型，因为这个key不存在，那么这个值会默认初始化为一个指针，但是其指向位置并未给定，因此直接使用会出错。
* 建议使用 `if v, ok := m[k]; ok {}` 使用方式来使用map，这一定是最标准，最严谨的写法，可以避免上一点提到的错误。
* map 通过 `make` 初始化，可以在可预估map中存放条目个数的情况下，对底层内存申请次数进行控制，进而减少开销，提升性能。可以对一些明确的场景指定这个 `cap` 值，这样是建议的写法。指定 `cap=10`，实际存放第11个数据时其可正常存放，通过 `make` 参数指定的只是其初始容量。
* map 并不支持通过`cap` 函数来计算其容量。这一点与 slice 是不同的。
* map 支持 `len` 函数来计算其中实际存放的条目个数。
* 可以通过 `delete(m, key)`来删除指定key。这时对应其长度也会减1。如果对一个不存在的 key 执行delete操作，则没有任何影响，也不会报错。
* 因为map是一个指针类型，因此其变量间赋值实际引用的数据区域是相同的，因此对及数据的修改，可以通过多个变量都可见，并且其结果是一致的。这里并没有提到map中的val是什么类型，与val的类型无关。
* 常规的 map 并不是协程安全的，如果要在多个协程中并发读写，需要使用 sync 包中的map操作，或者是借助 sync 包中的锁机制来实现。

```golang
func main() {
    fmt.Println("Hello World")

    m := make(map[int]bool, 2)
    fmt.Println(len(m))

    m[111] = true
    m[222] = false
    fmt.Println(len(m))

    m2 := m

    delete(m, 333)
    m[110] = true
    m[221] = false
    m[112] = true
    m[223] = false
    m[114] = true
    m[226] = false

    m2[333] = true
    fmt.Println(len(m), len(m2))
}
```

```plain
// OUTPUT
Hello World
0
2
9 9
```

## slice

* slice 是可变数组。在实际业务中基本不会直接使用数组，以至于要初始化一个数组还需要查阅文档。^_^
* slice 的初始化也是通过 `make([]int, len, cap)` 其中 `cap`  参数可以省略，但是 `len`参数必须指定。
* 未初始化的 slice 为nil, 可对其执行 append 操作。但是执行赋值、取值操作则会出现panic错误。
* slice 的切片操作语法 `a[low : high]` ，其结果包括low索引，但是不包括high 索引，即其截取结果的长度为 `high - low`。
* 在已经初始化的 slice 上执行赋值取值操作，索引不能超过其最大长度减一。
* 在已经初始化的 slice 上执行切片操作，其索引可以超过其最大长度，但是不可以超过其最大cap值。如果初始化时没有指定cap，cap规则失效，以其最大长度为准。
* slice 也是属于指针类型，因此变量间传递，对其进行修改操作，在左右引用上都是可见的，且是一致的。如果要对其值复制，可以通过 `for` 循环来对其每个元素赋值来实现“深拷贝”。
* 对slice 执行切片操作生成的新的 slice 其值默认会跟随原始 slice 变化，但是当其中任一个slice的长度发生变化时可能导致底层内存重新分配，这时其就会脱离关系。当长度有多大变化时会出现这种情况，并不能准确说明，及时可以准确说明也不建议使用这些隐含的规则，容易在后期的调整时造成代码bug。
* slice 也不是协程安全的数据类型，如果在多协程中使用，需要借助 sync 包中的锁机制或者 channel 来实现。
* 对 slice 执行 append 操作，原来的 slice 并不会被修改，而是重新生成新的slice。但是如果原有slice 容量足够时，其会使用其预留的内存空间，这样其相同的索引部分，其内存地址相同，因此其值相同，对其进行修改也是均可见，且是一致的。

```golang
func main() {
    fmt.Println("Hello World")

    var s []int = make([]int, 1, 5)

    s1 := s[:2]
    // t1 := s[2]
    t1 := s[0]
    fmt.Println(s, s1, t1)
}
```

```plain
// OUTPUT

Hello World
[0] [0 0] 0
```

```golang
func main() {
    fmt.Println("Hello World")

    var s []int = make([]int, 1, 5)

    s1 := s[:2]
    s2 := append(s, 2, 3, 4)
    s[0] = 9
    fmt.Println(s, s1, s2)
}

```

```plain
// OUTPUT

Hello World
[9] [9 2] [9 2 3 4]
```

## channel

* channel 也是 Golang 中的核心部分，不过也算是一个高级用法，用来解决多个goroutine 通信的问题。即：不要通过共享变量来通信，而是通过通信来共享变量。其中就是意指通过channel 来实现其变量共享的问题。
* channel 虽然有用，好用，但是使用频率远远低于 slice / map ，因此包括很多 golang 工程师知道其概念，但是对其详细用法依然不能熟练应用。每次使用时，还需要参考文档方可进行。我觉的这也并不是啥问题，不过晋升或者面试时这个应该是必须要提到的，因此在需要时补补功课也是很关键的。
* chanel 通过 `<-`符号进行读写操作。不管是读，还是写，这个箭头方向永远是朝左。如果 channel 变量名在该符号的右侧，表示这是一个读操作；如果 channel 变量名在该符号的左侧则表示这是一个写操作。
* channel 初始化使用make。如果使用 `make(chan int, buffsize)`可以创建带缓存的channel，否则默认通过 `var c chan int` 创建的 channel 其值为nil, 对其进行写入、读取操作一致处于阻塞状态，但并不会出现panic，因此要特别留意，这里使用不当容易产生奇怪的bug, 同时也建议养成使用make的好习惯。
* 无缓存的 channel , 当执行写入时，需要等到此消息被其他协程消费后才返回，否则一致处于阻塞状态。
* 有缓存的 channel，当在缓存区有可用空间时，这是执行写入，写入的协程会立即返回。如果没有协程消费 channel 中的数据，继续写入，缓存区会填满，这是也会进入阻塞状态。如果场景运行，尽可能使用有缓存区的 channel，提升系统吞吐能力。
* 对于未初始化的 channel，执行 close 操作则出现panic错误。对与已经初始化的channel 执行多次 close 也同样会出现panic错误。
* 对于已经关闭的 channel 执行 send 操作则出现 panic 错误。
* 对于已经关闭的 channel 执行 read 操作并不会出错，而是返回默认值，也不再阻塞。
* 对于正在等待接收 channel 消息的协程，如果这时另一个协程执行了close 操作，则当前协程会理解收到一个消息，其值为 channel 中数据类型的默认值，ok 为false。如果channel 未关闭，则 ok 为true。可利用这个状态来识别是否是 close 操作发送的消息。

```golang
func main() {
    fmt.Println("Hello World")

    var d = make(chan int)

    go func() {
        <-d
        fmt.Println("go1")
    }()

    go func() {
        d <- 1
        fmt.Println("go2")
    }()

    time.Sleep(20 * time.Second)
}
```

```plain
// OUTPUT

Hello World
go1
go2
```

```plain
// OUTPUT

Hello World
go2
go1

```

```golang

func main() {
    fmt.Println("Hello World")

    var c chan int
    var d = make(chan int)

    if c == nil {
        fmt.Println("sss")
    }

    go func() {
        <-d
        <-c
        fmt.Println("go1")
    }()

    go func() {
        d <- 1
        c <- 1
        fmt.Println("go2")
    }()

    time.Sleep(20 * time.Second)
}
```

```plain
OUTPUT

Hello World
sss
```

```golang
func main() {
    fmt.Println("Hello World")

    var d = make(chan int)

    go func() {
        d <- 1
        // close(d)
        fmt.Println("go2")
    }()

    go func() {
        for {
            select {
            case a, ok := <-d:
                fmt.Println("go1", a, ok)

            }
        }

    }()

    time.Sleep(20 * time.Second)
}
```

```plain
// OUTPUT

Hello World
go1 1 true
go2
```
