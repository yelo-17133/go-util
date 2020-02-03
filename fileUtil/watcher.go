// ------------------------------------------------------------------------------
// 文件监控器，用来实现 “监控某个文件的变化，如果文件被修改则触发回调通知”
// ------------------------------------------------------------------------------
package fileUtil

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

func LoadAndWatchFile(path string, onChange func(data []byte, err error) error, d time.Duration) (*time.Ticker, error) {
	lastMod, lastToken, data, err := loadFile(path, 0, "")
	if onChange != nil && (data != nil || err != nil) {
		err = onChange(data, err)
	}
	if err == nil && d > 0 {
		ticker := time.NewTicker(d)
		go func() {
			for range ticker.C {
				lastMod, lastToken, data, err = loadFile(path, lastMod, lastToken)
				if onChange != nil && (data != nil || err != nil) {
					onChange(data, err)
				}
			}
		}()
		return ticker, nil
	}
	return nil, err
}

func loadFile(path string, lastMod int64, lastToken string) (int64, string, []byte, error) {
	// 检查文件有没有修改过
	osFile, err := os.Stat(path)
	if err != nil {
		return lastMod, lastToken, nil, fmt.Errorf("无法访问文件 %v", path)
	}

	// 检查文件最后修改时间有没有发生变化
	if osFile.ModTime().UnixNano() <= lastMod {
		return lastMod, lastToken, nil, nil
	} else {
		lastMod = osFile.ModTime().UnixNano()
	}

	// 检查文件内容有没有发生变化
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return lastMod, lastToken, nil, err
	}
	tokenStr := fmt.Sprintf("%x", md5.Sum(b))
	if lastToken == tokenStr {
		return lastMod, lastToken, nil, nil
	}

	return lastMod, tokenStr, b, nil
}
