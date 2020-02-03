package osUtil

import (
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
)

var (
	once  sync.Once
	onSig = make([]func(os.Signal), 0, 16)
)

// 当收到操作系统的退出信号（SIGHUP、SIGINT、SIGQUIT、SIGABRT、SIGKILL、SIGUSR1、SIGUSR2、SIGTERM）时触发
func OnSignalExit(f func(sig os.Signal)) {
	if f != nil {
		once.Do(func() {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGABRT, syscall.SIGKILL, syscall.Signal(10), syscall.Signal(12), syscall.SIGTERM)
			go func() {
				sig := <-sigs
				for _, f := range onSig {
					func() {
						defer func() {
							if e := recover(); e != nil {
								debug.PrintStack()
							}
						}()
						f(sig)
					}()
				}
				os.Exit(sigCode(sig))
			}()
		})
		onSig = append(onSig, f)
	}
}

func sigCode(sig os.Signal) int {
	if sig != nil {
		switch sig.String() {
		case "hangup":
			return 1
		case "interrupt":
			return 2
		case "quit":
			return 3
		case "illegal instruction":
			return 4
		case "trace/breakpoint trap":
			return 5
		case "aborted":
			return 6
		case "bus error":
			return 7
		case "floating point exception":
			return 8
		case "killed":
			return 9
		case "user defined signal 1":
			return 10
		case "segmentation fault":
			return 11
		case "user defined signal 2":
			return 12
		case "broken pipe":
			return 13
		case "alarm clock":
			return 14
		case "terminated":
			return 15
		}
	}
	return -1
}
