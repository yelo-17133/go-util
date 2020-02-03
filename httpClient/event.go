package httpClient

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

// ------------------------------------------------------------------------------ Request
var requestHandler []func(*Client, error, float32)

// 当 HTTP 请求执行完毕之后触发
func OnRequest(f func(c *Client, err error, took float32)) {
	if f != nil {
		requestHandler = append(requestHandler, f)
	}
}

func fireRequest(c *Client, err error, took float32) {
	defer func() {
		if e := recover(); e != nil {
			os.Stderr.WriteString(fmt.Sprintf("[%v] httpClient panic: %v\n", time.Now().Format("2006-01-02 15:04:05.000"), e))
			debug.PrintStack()
		}
	}()
	for _, f := range requestHandler {
		f(c, err, took)
	}
}
