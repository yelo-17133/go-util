package _utilTest

import "fmt"
import "strings"

var (
	MySqlSource   string
	RedisAddr     string
	RedisPassword string
	EveApiProxy   string
	KafkaBrokers  string
)

func Init() {
	InitEnv("dev")
}

func InitEnv(env string) {
	switch strings.ToLower(env) {
	case "dev":
		MySqlSource = "root:123456@tcp(172.27.1.21:3306)/?charset=utf8&timeout=3s&parseTime=true&loc=Local"
		RedisAddr = "172.27.1.21:6379"
		RedisPassword = "vp7sb_SxO0Jk#v%hwIb2YX84"
		EveApiProxy = "http://172.27.1.21:8000"
		KafkaBrokers = "172.27.1.38:19092,172.27.1.38:29092,172.27.1.38:39092"
	case "test":
		MySqlSource = "root:123456@tcp(172.27.1.54:3306)/?charset=utf8&timeout=3s&parseTime=true&loc=Local"
		RedisAddr = "172.27.1.54:6379"
		RedisPassword = "vp7sb_SxO0Jk#v%hwIb2YX84"
		EveApiProxy = "http://172.27.1.54:8000"
		KafkaBrokers = "172.27.1.54:19092,172.27.1.54:29092,172.27.1.54:39092"
	default:
		panic(fmt.Errorf("invalid env(%v)", env))
	}
}
