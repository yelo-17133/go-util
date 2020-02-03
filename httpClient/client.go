// ------------------------------------------------------------------------------
// 对 HTTP Client 的进一步封装
//   1、简化各种参数的设置
//   2、针对返回 Json 结构的 Restful Api，封装了更加简化的接口
//   3、内部实现自动重试机制，调用接口时如果发生（除超时之外的）错误则自动重试，以消除服务抖动的问题
// ------------------------------------------------------------------------------
package httpClient

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/emirpasic/gods/lists/singlylinkedlist"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"go-util/arrUtil"
	"go-util/jsonUtil"
	"go-util/strUtil"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// HTTP 客户端类，用户封装 HTTP 请求
type Client struct {
	*http.Client
	fromPool     bool              // 是否是连接池中的连接
	method       string            // http method
	url          string            // http url
	body         interface{}       // http post body
	headers      map[string]string // http headers
	cookie       []*http.Cookie    // cookie
	ctx          interface{}       // context
	ignoreEvents bool
	idle         bool
	retry        int
	retryInterel time.Duration
	err          error
	responseData []byte
	responseText string

	StatusCode int
	Status     string
}

type RequestParam struct {
	Method string            `json:"method,omitempty"`
	Host   string            `json:"host,omitempty"`
	Api    string            `json:"api,omitempty"`
	Header map[string]string `json:"header,omitempty"`
	Body   interface{}       `json:"body,omitempty"`
	Cookie []string          `json:"cookie,omitempty"`
}

var (
	DefaultRetry        = 3
	DefaultRetryIntevel = 100 * time.Millisecond
	DefaultContentType  = ""
	defaultKeepAlive    = 120 * time.Second

	idleClients        = singlylinkedlist.New() // 空闲的 client 列表，用来管理空闲连接
	requestPathPattern = regexp.MustCompile("://.*?(/[^?#]+)")
	asciiPattern       = regexp.MustCompile("^[\\x00-\\xff]+$")
	lock               sync.Mutex
)

func SetDefaultTimeout(timeout time.Duration, keepAlive ...time.Duration) {
	var keepAliveVal time.Duration
	if len(keepAlive) != 0 && keepAlive[0] <= 0 {
		keepAliveVal = keepAlive[0]
	} else {
		keepAliveVal = 120 * time.Second
	}
	defaultKeepAlive = keepAliveVal

	http.DefaultClient.Timeout = timeout
	transport := http.DefaultTransport.(*http.Transport)
	transport.ResponseHeaderTimeout = timeout
	transport.DialContext = (&net.Dialer{
		Timeout:   timeout,
		KeepAlive: keepAliveVal,
	}).DialContext
}

// ------------------------------------------------------------------------------ getter & setter
// （从空闲的连接池中）打开一个 HTTP 客户端连接。请在使用完毕之后调用 Close 释放资源。
func Open() *Client {
	var client *Client

	lock.Lock()
	defer lock.Unlock()

	it := idleClients.Iterator()
	for it.Next() {
		if client := it.Value().(*Client); client.idle {
			client.Reborn().idle = false
			return client
		}
	}

	client = (&Client{Client: http.DefaultClient, fromPool: true}).Reborn().setIdle(false)
	idleClients.Add(client)

	return client
}

func (this *Client) setIdle(v bool) *Client {
	this.idle = v
	return this
}

// 关闭 HTTP 客户端连接（归还给连接池）。
func (this *Client) Close() {
	if this.fromPool {
		lock.Lock()
		defer lock.Unlock()
	}
	this.idle = true
}

// 关闭 HTTP 客户端连接（归还给连接池），同时返回上一次 Request 返回的结果 (ResponseData, error)
func (this *Client) CloseAndResult() ([]byte, error) {
	result, err := this.responseData, this.err
	this.Close()
	return result, err
}

// 关闭 HTTP 客户端连接（归还给连接池），同时返回上一次 Request 过程中发生的错误
func (this *Client) CloseAndError() error {
	err := this.err
	this.Close()
	return err
}

// 重置所有参数（但不关闭连接），这样可以连续多次使用。
func (this *Client) Reborn() *Client {
	this.ctx = nil
	this.method = ""
	this.url = ""
	this.body = nil
	this.headers = make(map[string]string)
	if DefaultContentType != "" {
		this.headers["Content-Type"] = DefaultContentType
	}
	this.retry = DefaultRetry
	this.retryInterel = DefaultRetryIntevel
	this.err = nil
	this.responseData = nil
	this.responseText = ""
	this.ignoreEvents = false
	this.Timeout = http.DefaultClient.Timeout
	return this
}

// 获取上下文对象
// 请在调用 Close（归还到连接池）方法之前调用，因为归还到连接池之后有可能由于被其他线程取出而被重置参数。
func (this *Client) Context() interface{} {
	return this.ctx
}

// 设置上下文对象
func (this *Client) SetContext(ctx interface{}) *Client {
	this.ctx = ctx
	return this
}

// 获取上一次请求的 Method
// 请在调用 Close（归还到连接池）方法之前调用，因为归还到连接池之后有可能由于被其他线程取出而被重置参数。
func (this *Client) Method() string {
	return this.method
}

func (this *Client) SetMethod(method string) *Client {
	this.method = strings.ToUpper(method)
	return this
}

// 获取上一次请求的 Url
// 请在调用 Close（归还到连接池）方法之前调用，因为归还到连接池之后有可能由于被其他线程取出而被重置参数。
func (this *Client) Url() string {
	return this.url
}

func (this *Client) SetUrl(v string) *Client {
	this.url = v
	return this
}

// 获取上一次请求的 RequestPath
// 请在调用 Close（归还到连接池）方法之前调用，因为归还到连接池之后有可能由于被其他线程取出而被重置参数。
func (this *Client) RequestPath() string {
	if arr := requestPathPattern.FindStringSubmatch(this.url); len(arr) == 2 {
		return arr[1]
	} else {
		return this.url
	}
}

// 设置超时参数。默认的超时参数可以通过 http.TimeFormat、http.DefaultTransport 全局设置
func (this *Client) SetTimeout(timeout time.Duration, keepAlive ...time.Duration) *Client {
	var keepAliveVal time.Duration
	if len(keepAlive) != 0 && keepAlive[0] <= 0 {
		keepAliveVal = keepAlive[0]
	} else {
		keepAliveVal = defaultKeepAlive
	}

	this.Timeout = timeout
	switch t := this.Transport.(type) {
	case *http.Transport:
		t.ResponseHeaderTimeout = timeout
		t.DialContext = (&net.Dialer{
			Timeout:   this.Timeout,
			KeepAlive: keepAliveVal,
		}).DialContext
	}
	return this
}

// 获取上一次请求的 Request Body
// 请在调用 Close（归还到连接池）方法之前调用，因为归还到连接池之后有可能由于被其他线程取出而被重置参数。
func (this *Client) Body() interface{} {
	return this.body
}

// 设置 Request Body
func (this *Client) SetBody(v interface{}) *Client {
	this.body = v
	return this
}

// 获取上一次请求的 Header
// 请在调用 Close（归还到连接池）方法之前调用，因为归还到连接池之后有可能由于被其他线程取出而被重置参数。
func (this *Client) Headers() map[string]string {
	return this.headers
}

func (this *Client) ClearHeader() *Client {
	this.headers = make(map[string]string)
	return this
}

// 设置 Header。 如果 key 是 camel 或者下划线分割的格式，则函数会将其转为 UpperKebab 格式（中划线分割的首字母大写单词）。
func (this *Client) SetHeader(key string, val interface{}) *Client {
	return this.setHeader(key, val)
}

// 设置 Header。 如果 key 是 camel 或者下划线分割的格式，则函数会将其转为 UpperKebab 格式（中划线分割的首字母大写单词）。
func (this *Client) SetHeaderMulti(dict map[string]interface{}) *Client {
	if dict != nil {
		for key, val := range dict {
			this.setHeader(key, val)
		}
	}
	return this
}

// 设置 Header。 如果 key 是 camel 或者下划线分割的格式，则函数会将其转为 UpperKebab 格式（中划线分割的首字母大写单词）。
func (this *Client) SetHeaderMultiS(dict map[string]string) *Client {
	if dict != nil {
		for key, val := range dict {
			this.setHeader(key, val)
		}
	}
	return this
}

func (this *Client) setHeader(key string, a interface{}) *Client {
	// 添加中划线，转换为标准格式
	key = strUtil.CamelToUpperKebab(key)
	val := ""
	if a != nil {
		switch t := a.(type) {
		case string:
			val = t
		case []byte:
			val = string(t)
		default:
			val = jsonUtil.MustMarshalToString(t)
		}
	}

	if val != "" && !asciiPattern.MatchString(val) {
		val = "Base64:" + base64.StdEncoding.EncodeToString([]byte(val))
	}

	if this.headers == nil {
		this.headers = map[string]string{key: val}
	} else {
		this.headers[key] = val
	}

	return this
}

func (this *Client) SetContentType(v string) *Client {
	this.setHeader("Content-Type", v)
	return this
}

func (this *Client) SetContentTypeJson() *Client {
	this.setHeader("Content-Type", "application/json; charset=UTF-8")
	return this
}

func (this *Client) AddCookie(a ...*http.Cookie) {
	this.cookie = append(this.cookie, a...)
}

// 获取上一次请求的请求参数
// 请在调用 Close（归还到连接池）方法之前调用，因为归还到连接池之后有可能由于被其他线程取出而被重置参数。
func (this *Client) RequestParams() *RequestParam {
	params := &RequestParam{Method: this.method, Api: this.url}
	if pos := strings.Index(this.url, "://"); pos != -1 {
		if v, err := url.Parse(this.url); err == nil {
			params.Host = v.Scheme + "://" + v.Host
			params.Api = this.url[len(params.Host):]
		}
	}
	if len(this.headers) != 0 {
		params.Header = make(map[string]string, len(this.headers))
		for k, v := range this.headers {
			params.Header[k] = v
		}
	}
	if len(this.cookie) != 0 {
		for _, v := range this.cookie {
			params.Cookie = append(params.Cookie, v.String())
		}
		sort.Strings(params.Cookie)
	}
	if this.body != nil && this.body != "" {
		params.Body = this.body
	}
	return params
}

// 设置是否忽略 HTTP 事件。 忽略后的请求将不会被发送给通过 OnRequest 设置的回调函数。默认不忽略。
func (this *Client) IgnoreEvents() *Client {
	this.ignoreEvents = true
	return this
}

// 设置重试策略
func (this *Client) SetRetry(retry int, intervel time.Duration) *Client {
	this.retry, this.retryInterel = retry, intervel
	return this
}

// （使用已经设置好的 Method、Url、Body、Header 等）发起 HTTP 请求，并等待处理结果。
func (this *Client) Request() *Client {
	begin := time.Now().UnixNano()
	this.doRequest()
	for i := 1; i < this.retry && (this.err != nil || this.StatusCode <= 0 || this.StatusCode >= 400); i++ {
		if this.err == nil {
			if this.StatusCode > 0 && this.StatusCode < 400 {
				// 如果调用成功，不重试
				break
			} else if this.StatusCode == 408 || this.StatusCode == 504 {
				// 如果是超时，不重试
				break
			}
		} else {
			if -1 != strings.Index(this.err.Error(), "timeout awaiting response headers") || -1 != strings.Index(this.err.Error(), "Client.Timeout exceeded while awaiting header") {
				// 如果是超时，不重试
				break
			}
		}
		time.Sleep(this.retryInterel)
		this.doRequest()
	}
	if !this.ignoreEvents {
		fireRequest(this, this.err, float32(time.Now().UnixNano()-begin)/1000000)
	}
	return this
}

// 获取上一次 Request 返回的结果，(ResponseData, error)
// 请在调用 Close（归还到连接池）方法之前调用，因为归还到连接池之后有可能由于被其他线程取出而被重置参数。
func (this *Client) Result() ([]byte, error) {
	return this.responseData, this.err
}

// 获取上一次 Request 过程中发生的错误
// 请在调用 Close（归还到连接池）方法之前调用，因为归还到连接池之后有可能由于被其他线程取出而被重置参数。
func (this *Client) Error() error {
	return this.err
}

// 获取上一次 Request 是否成功（未发生错误，且状态码符合预期）
// 如果参数 status 为空，则判断状态码是否介于 [200, 400) 区间
// 请在调用 Close（归还到连接池）方法之前调用，因为归还到连接池之后有可能由于被其他线程取出而被重置参数。
func (this *Client) Success(status ...int) (bool, error) {
	if this.err != nil {
		return false, this.err
	}
	if len(status) == 0 {
		if this.StatusCode < 200 || this.StatusCode >= 400 {
			return false, errors.New(this.Status)
		}
	} else {
		if this.StatusCode != 0 && arrUtil.IndexOfInt(status, this.StatusCode) == -1 {
			return false, errors.New(this.Status)
		}
	}
	return true, nil
}

// 获取上一次 Request 返回的 ResponseData
// 请在调用 Close（归还到连接池）方法之前调用，因为归还到连接池之后有可能由于被其他线程取出而被重置参数。
func (this *Client) ResponseData() []byte {
	return this.responseData
}

// 获取上一次 Request 返回的 ResponseText
// 请在调用 Close（归还到连接池）方法之前调用，因为归还到连接池之后有可能由于被其他线程取出而被重置参数。
func (this *Client) ResponseText() string {
	if this.responseText == "" {
		this.responseText = string(this.responseData)
	}
	return this.responseText
}

func (this *Client) doRequest() ([]byte, error) {
	this.err = nil
	this.responseData = nil
	this.responseText = ""
	this.StatusCode = -1
	this.Status = "Send Request Error"

	if this.method == "" {
		this.method = "GET"
	}

	var body io.Reader = nil
	if this.body != nil {
		switch t := this.body.(type) {
		case string:
			body = strings.NewReader(t)
		case []byte:
			body = bytes.NewReader(t)
		default:
			data, err := jsonUtil.Marshal(this.body)
			if err != nil {
				this.err = fmt.Errorf("body 序列化失败: %v", err)
				return this.responseData, this.err
			}
			body = bytes.NewReader(data)
		}
	}

	req, err := http.NewRequest(this.method, this.url, body)
	if err != nil {
		this.err = err
		return this.responseData, this.err
	}

	if len(this.headers) != 0 {
		for k, v := range this.headers {
			req.Header.Set(k, v)
		}
	}
	if len(this.cookie) != 0 {
		for _, v := range this.cookie {
			req.AddCookie(v)
		}
	}

	resp, err := this.Do(req)
	if err != nil {
		this.err = err
		return this.responseData, this.err
	}
	defer resp.Body.Close()

	// read response
	this.responseData, this.err = ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 0 {
		this.Status = resp.Status
		this.StatusCode = resp.StatusCode
		if this.StatusCode < 200 || this.StatusCode >= 400 {
			this.err = errors.New(this.Status)
		}
	}
	return this.responseData, err
}

// 使用 POST 方式发起 HTTP 请求并等待处理结果。 等同于 SetMethod("POST").Request()
func (this *Client) Post() *Client {
	return this.SetMethod("POST").Request()
}

// 使用 GET 方式发起 HTTP 请求并等待处理结果。 等同于 SetMethod("GET").Request()
func (this *Client) Get() *Client {
	return this.SetMethod("GET").Request()
}

// （使用已经设置好的 Method、Url、Body、Header 等）发起 HTTP 请求，并尝试将返回的 ResponseData 按照 Json 格式反序列化到参数指定的对象中。
// 如果 Json 反序列化出错，则可以通过 error() 获取错误对象。
func (this *Client) RequestJson(obj interface{}) *Client {
	this.Request()
	if this.err == nil && obj != nil {
		err := jsonUtil.Unmarshal(this.responseData, obj)
		if err != nil {
			this.err = fmt.Errorf("json decode error, responseText: %s, err=%v", this.ResponseText(), err)
		}
	}
	return this
}

// 使用 POST 方式发起 HTTP 请求，并尝试将返回的 ResponseData 按照 Json 格式反序列化到参数指定的对象中。 等同于 SetMethod("POST").RequestJson()
func (this *Client) PostJson(obj interface{}) *Client {
	return this.SetMethod("POST").RequestJson(obj)
}

// 使用 GET 方式发起 HTTP 请求，并尝试将返回的 ResponseData 按照 Json 格式反序列化到参数指定的对象中。 等同于 SetMethod("GET").RequestJson()
func (this *Client) GetJson(obj interface{}) *Client {
	return this.SetMethod("GET").RequestJson(obj)
}

func (this *RequestParam) SetHost(v string) *RequestParam {
	this.Host = v
	return this
}
