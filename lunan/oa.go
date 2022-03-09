package lunan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

var authEndpoint = ""
var client = http.DefaultClient
var pool sync.Pool

func init() {
	//初始化一个pool
	pool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, 8192))
		},
	}
	authEndpoint = os.Getenv("LUNAN_AUTH_ENDPOINT")
	if authEndpoint == "" {
		log.Println("若想使用公司密码认证服务, 请输入公司认证API, 否则无法使用密码认证功能")
	}
}
func AuthEnabled() bool {
	return authEndpoint == ""
}

type AuthResponse struct {
	Error    string        `json:"error"`
	ErrMsg   string        `json:"errmsg"`
	Data     []interface{} `json:"data"`
	UserName string        `json:"UserName"`
	BH       string        `json:"BH"`
	Cost     int           `json:"cost"`
}

// Auth 用在 object/check.go:156
func Auth(ctx context.Context, username, password string) error {
	//获取一个新的，如果不存在则会调用new创建
	buffer := pool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
		if buffer != nil {
			//重新放回去
			pool.Put(buffer)
			buffer = nil
		}

	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authEndpoint, strings.NewReader(url.Values{"Name": []string{username}, "Psd": []string{password}}.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var authResp AuthResponse
	err = json.Unmarshal(data, &authResp)
	if err != nil {
		return err
	}
	if authResp.Error != "" {
		// 确保数据库端是正确返回
		if strings.Contains(authResp.Error, "密码不正确") {
			return errors.New(fmt.Sprintf("lunan user name or password incorrect, provide %v with %v", username, password))
		} else {
			return errors.New(authResp.Error)
		}
	}
	return nil
}
