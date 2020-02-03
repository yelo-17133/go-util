package runtimeUtil

import (
	"runtime"
	"strings"
)

type Frame struct {
	Func string `json:"func"`
	File string `json:"file"`
	Line int    `json:"line"`
}

type StackOptions struct {
	FormatFunc func(f *Frame)
}

func Stack() []*Frame {
	return StackFilter(3, nil)
}

func PanicStack() []*Frame {
	panicFound := false
	return StackFilter(3, func(f *Frame) bool {
		if !panicFound {
			if f.Func == "runtime.gopanic" {
				panicFound = true
				return false
			}
		} else if strings.HasPrefix(f.Func, "runtime.panic") {
			return false
		}
		return panicFound
	})
}

func StackFilter(skip int, filter func(f *Frame) bool) []*Frame {
	pc := make([]uintptr, 64)
	n := runtime.Callers(skip, pc)
	if n <= 1 {
		return nil
	}
	pc = pc[:n-1]
	frames := runtime.CallersFrames(pc)
	arr := make([]*Frame, 0, n)
	for i := 0; ; i++ {
		f, more := frames.Next()
		if f.Func != nil {
			v := &Frame{Func: f.Function, File: f.File, Line: f.Line}
			if filter == nil || filter(v) {
				arr = append(arr, v)
			}
		}
		if !more {
			break
		}
	}
	return arr
}
