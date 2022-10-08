package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/armon/go-socks5"
	"github.com/elazarl/goproxy"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var privateIPBlocks []*net.IPNet

func ipInit() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func getClientIp() ([]string, error) {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		return nil, err
	}

	var lst []string
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !isPrivateIP(ipnet.IP) {
			if ipnet.IP.To4() != nil {
				ip := ipnet.IP.String()
				skip := false
				for _, i := range lst {
					if i == ip {
						skip = true
					}
				}
				if !skip {
					lst = append(lst, ip)
				}
			}
		}
	}

	return lst, nil
}

type IP struct {
	Query string
}

func getip2() string {
	req, err := http.Get("http://ip-api.com/json/")
	if err != nil {
		return ""
	}
	defer req.Body.Close()

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return ""
	}

	var ip IP
	json.Unmarshal(body, &ip)

	return ip.Query
}

func main() {
	ipInit()
	ttl := flag.Int64("time", 25*60, "存活时长，单位秒，默认为25分钟")
	user := flag.String("user", "", "用户名，默认为空")
	password := flag.String("password", "", "密码，默认为空")
	port := flag.Int64("port", 13325, "端口号")
	portHttp := flag.Int64("port-http", 13326, "HTTP代理端口号")
	flag.Parse()

	fmt.Println("http/socks5简易工具©sealdice.com")
	fmt.Printf("将在服务器上开启一个socks5服务，端口%d，默认持续时长为25分钟\n", *port)
	fmt.Printf("将在服务器上开启一个http代理服务，端口%d，默认持续时长为25分钟\n", *portHttp)

	ip, err := getClientIp()
	if err != nil {
		return
	}

	_ip := getip2()
	if _ip != "" {
		ip = append(ip, _ip)
	}

	publicIP := strings.Join(ip, ", ")
	if publicIP == "" {
		fmt.Println("\n未检测到公网IP")
	} else {
		fmt.Println("\n可能的公网IP: ", publicIP)
	}
	fmt.Println("请于服务器管理面板放行你要用的端口(一般为13326即http)，协议TCP")
	fmt.Println("如果是Windows Server 2012R2及以上系统，请再额外关闭系统防火墙或设置规则放行")
	fmt.Println()

	go func() {
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

		address := "0.0.0.0:" + strconv.FormatInt(*port, 10)
		fmt.Println("正在启动服务:", address)
		if err := server.ListenAndServe("tcp", address); err != nil {
			panic(err)
		}
	}()

	func() {
		address := "0.0.0.0:" + strconv.FormatInt(*portHttp, 10)
		fmt.Println("正在启动服务:", address)
		proxy := goproxy.NewProxyHttpServer()
		proxy.Verbose = true
		log.Fatal(http.ListenAndServe(address, proxy))
	}()
}
