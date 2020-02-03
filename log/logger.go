package log

type Logger interface {
	Error(format string, a ...interface{})
	Warn(format string, a ...interface{})
	Info(format string, a ...interface{})
	Debug(format string, a ...interface{})
}

type AttrLogger interface {
	Logger
	With(name string, val interface{}) AttrLogger
	WithMulti(map[string]interface{}) AttrLogger
}

type FlushLogger interface {
	Logger
	Flush()
}

type CloseLogger interface {
	Logger
	Close()
}
