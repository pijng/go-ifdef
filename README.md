# go-ifdef
Add trivial `#ifdef`, `#else` and `#define` macros to your go code.

`go-ifdef` has built-in support for GOOS in form of `#ifdef GOOS:<target>`
You can use any valid `GOOS` value.

### Usage

1. Install `go-ifdef` with:

```bash
go install github.com/pijng/go-ifdef@latest
```

2. Build your project with `go build` while specifying go-ifdef preprocessor:

```bash
go build -o output -a -toolexec="go-ifdef <absolute/path/to/project>" main.go
```

**Important:**
  * `-a` flag is required to recompile all your project, otherwise go compiler might do nothing and use cached build
  * `<absolute/path/to/project>` is and absolute path to the root of your project. If you run `go build` from the root – simply specify `$PWD` as an argument.

3. Run the final binary:

```bash
./output
```

### Demonstration

Suppose we have this code:

```go
package main

import "fmt"

func main() {
	var i int

	// #ifdef GOOS:darwin
	i = 100
	// #else
	i = 727
	// #endif

	fmt.Println(i)
}
```

If we compile it for MacOS and apply `go-ifdef` as a preprocessor, then we'll get the following result:

```bash
$ env GOOS=darwin go build -a -toolexec="go-ifdef $PWD" main.go
$ ./main
100
$
```

But if we change the directive to `linux`, for example like this:

```go
package main

import "fmt"

func main() {
	var i int

	// #ifdef GOOS:linux
	i = 100
	// #else
	i = 727
	// #endif

	fmt.Println(i)
}
```

and then compile it to the same MacOS, then we'll get different result:


```bash
$ env GOOS=darwin go build -a -toolexec="go-ifdef $PWD" main.go
$ ./main
727
$
```


Or another example with conditional struct populating:

```go
package main

import "fmt"

func main() {
	fmt.Printf("%+v\n", someResult())
}

type result struct {
	os    string
	items []int
}

func someResult() result {
	res := result{os: "Not windows"}
	res.items = append(res.items, []int{1, 2, 3}...)

	// #ifdef GOOS:windows
	res.os = "windows!"
	res.items = append(res.items, []int{4, 5}...)
	// #endif

	return res
}
```

If we compile this code for a platform other than Windows, then upon execution, we will get the following:

```bash
$ env GOOS=linux go build -o main -a -toolexec="go-ifdef $PWD" main.go
$ ./main
{os:Not windows items:[1 2 3]}
$
```

But if we compile it for Windows, we'll get this instead:

```bash
$ env GOOS=windows go build -o main -a -toolexec="go-ifdef $PWD" main.go
$ ./main
{os:windows! items:[1 2 3 4 5]}
$
```
___

### Custom directives

You can define custom directives with `#define` keyword.
⚠️ **Only boolean values are supported**

For example:

```go
package main

import "fmt"

// #define DEBUG true

func main() {
	fmt.Printf("%d\n", someResult())
}

func someResult() int {
	// #ifdef DEBUG
	fmt.Println("DEBUGGING")
	// #else
	fmt.Println("NOT DEBUGGING")
	// #endif

	return 0
}
```

If we compile it and run:

```bash
$ go build -o main -a -toolexec="go-ifdef $PWD" main.go
$ ./main
DEBUGGING
0
$
```

But if we change `#define DEBUG` to false:

```go
package main

import "fmt"

// #define DEBUG false

func main() {
	fmt.Printf("%d\n", someResult())
}

func someResult() int {
	// #ifdef DEBUG
	fmt.Println("DEBUGGING")
	// #else
	fmt.Println("NOT DEBUGGING")
	// #endif

	return 0
}
```

Then the result will be different:

```bash
$ go build -o main -a -toolexec="go-ifdef $PWD" main.go
$ ./main
NOT DEBUGGING
0
$
```