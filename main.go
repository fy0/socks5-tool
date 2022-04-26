package main

import (
	"flag"
	"fmt"
	"github.com/armon/go-socks5"
	"os"
	"strconv"
	"time"
)

func main() {
	ttl := flag.Int64("time", 15*60, "存活时长，单位秒，默认为15分钟")
	user := flag.String("user", "", "用户名，默认为空")
	password := flag.String("password", "", "密码，默认为空")
	port := flag.Int64("port", 13325, "端口号")
	flag.Parse()

	fmt.Println("Socks5简易工具 ©sealdice.com")
	fmt.Println("此工具将在服务器上开启一个socks5服务，端口13325，默认持续时长为15分钟[限时是为了安全]")

	secure := []socks5.Authenticator{}

	if *user != "" && *password != "" {
		fmt.Println("当前用户", *user)
		fmt.Println("当前密码", *password)

		cred := socks5.StaticCredentials{
			*user: *password,
		}
		cator := socks5.UserPassAuthenticator{Credentials: cred}
		secure = append(secure, cator)
	}

	// Create a SOCKS5 server
	conf := &socks5.Config{
		AuthMethods: secure,
	}
	server, err := socks5.New(conf)
	if err != nil {
		panic(err)
	}

	if *ttl > 0 {
		go func() {
			time.Sleep(time.Second * time.Duration(*ttl))
			fmt.Println("自动退出")
			os.Exit(0)
		}()
	}

	// Create SOCKS5 proxy on localhost port 8000
	address := "0.0.0.0:" + strconv.FormatInt(*port, 10)
	fmt.Println("正在启动服务:", address)
	if err := server.ListenAndServe("tcp", address); err != nil {
		panic(err)
	}
}
