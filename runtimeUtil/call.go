package runtimeUtil

func CallFunc(f func()) (panic interface{}) {
	defer func() {
		if e := recover(); e != nil {
			panic = e
		}
	}()
	f()
	return nil
}

func CallErrFunc(f func() error) (err error, panic interface{}) {
	defer func() {
		if e := recover(); e != nil {
			panic = e
		}
	}()
	return f(), nil
}
