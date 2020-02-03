package strUtil

import (
	"fmt"
	"strings"
)

type StringBuilder struct {
	data     []string
	index    int
	capacity int
}

func NewStringBuilder(str ...string) *StringBuilder {
	return (&StringBuilder{}).Append(str...)
}

func (this *StringBuilder) Empty() bool { return this.index == 0 }

func (this *StringBuilder) Clear() { this.index = 0 }

func (this *StringBuilder) Append(str ...string) *StringBuilder {
	if n := len(str); n > 0 {
		this.prepareAppend(n)
		for _, s := range str {
			this.data[this.index] = s
			this.index++
		}
	}
	return this
}

func (this *StringBuilder) Appendf(format string, a ...interface{}) *StringBuilder {
	return this.Append(fmt.Sprintf(format, a...))
}

func (this *StringBuilder) Reset() *StringBuilder {
	this.index = 0
	return this
}

func (this *StringBuilder) String() string {
	return strings.Join(this.data, "")
}

func (this *StringBuilder) prepareAppend(n int) {
	if n2 := this.index + n; n2 > this.capacity {
		for n2 > this.capacity {
			if this.capacity == 0 {
				this.capacity = 8
			} else {
				this.capacity *= 2
			}
		}
		if this.index == 0 {
			this.data = make([]string, this.capacity)
		} else {
			newData := make([]string, this.capacity)
			copy(newData, this.data)
			this.data = newData
		}
	}
}
