# For Golang application create pid file 

## Example:
```go
package main

import (
	"fmt"
	"os"

	"github.com/Wang/pid"
)

func main() {
	pidValue, err := pid.Create("my.pid")
	if err != nil {
		fmt.Errorf("create pid:%s", err.Error())
		os.Exit(1)
	}
	fmt.Printf("my pid:%d\n", pidValue)

}


```