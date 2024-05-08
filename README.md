# go-ifdef
Add trivial `#ifdef` and `#else` directives based on `GOOS` to your go code.

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

You can use any valid `GOOS` value.

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
