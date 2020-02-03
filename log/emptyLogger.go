package log

func EmptyLogger() Logger { return &emptyLogger{} }

type emptyLogger struct{}

func (*emptyLogger) Error(format string, a ...interface{}) {}

func (*emptyLogger) Warn(format string, a ...interface{}) {}

func (*emptyLogger) Info(format string, a ...interface{}) {}

func (*emptyLogger) Debug(format string, a ...interface{}) {}
