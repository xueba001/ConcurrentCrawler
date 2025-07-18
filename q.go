package main

//Go 的 import 必须显式列出所有用到的包，不能像 Python 那样动态导入，也不能像 Java 那样用 * 通配符。
import (
	"bufio" // 导入bufio包：用于缓冲IO操作
	"bytes" // 导入bytes包：用于字节切片操作
	"context" // 导入context包：用于上下文控制（如取消、超时）
	"flag" // 导入flag包：用于命令行参数解析
	"fmt" // 导入fmt包：用于格式化输出
	"io" // 导入io包：用于IO操作
	"log" // 导入log包：用于日志输出
	"net/http" // 导入net/http包：用于HTTP请求
	"os" // 导入os包：用于操作系统功能
	"os/signal" // 导入os/signal包：用于信号处理
	"path/filepath" // 导入path/filepath包：用于文件路径操作
	"strings" // 导入strings包：用于字符串操作
	"sync" // 导入sync包：用于并发同步
	"syscall" // 导入syscall包：用于系统调用
	"time" // 导入time包：用于时间相关操作
)

//2. 变量声明与全局变量
var (
	threads  int // 线程数：并发请求的最大数量
	interval int // 间隔：每轮请求之间的时间间隔（秒）
)

//nit() 是 Go 的特殊函数，每个包可以有多个，自动在 main 之前执行，适合做初始化
func init() {
	flag.IntVar(&threads, "t", 1, "并发线程数") // 解析-t参数，设置并发线程数，默认20
	flag.IntVar(&interval, "i", 1, "请求间隔(秒)") // 解析-i参数，设置请求间隔，默认1秒
	flag.Parse() // 解析命令行参数
}
//Go 为什么这样写：
//Go 用 type StructName struct {} 定义结构体，字段类型在后，支持嵌套和组合。
//Go 没有 class 关键字，结构体+方法实现面向对象。
type RequestTemplate struct {
	Name    string            // 模板名称
	Method  string            // HTTP方法（GET、POST等）
	URL     string            // 请求URL
	Headers map[string]string // 请求头部
	Body    []byte            // 请求体内容
}

// 解析原始HTTP请求内容，返回RequestTemplate结构体
func parseRawRequest(data []byte) (*RequestTemplate, error) {
	reader := bufio.NewReader(bytes.NewReader(data)) // 创建带缓冲的读取器
	firstLine, err := reader.ReadString('\n') // 读取请求行
	if err != nil {
		return nil, fmt.Errorf("读取请求行失败: %w", err) // 读取失败返回错误
	}
	firstLine = strings.TrimSpace(firstLine) // 去除首尾空白字符
	parts := strings.SplitN(firstLine, " ", 3) // 按空格分割请求行，最多3段
	if len(parts) < 2 {
		return nil, fmt.Errorf("请求行格式错误") // 检查请求行格式
	}
	method := parts[0] // 获取HTTP方法
	path := parts[1] // 获取请求路径

	headers := make(map[string]string) // 创建头部map
	for {
		line, err := reader.ReadString('\n') // 读取一行头部
		if err != nil || line == "\r\n" || line == "\n" {
			break // 读到空行或出错则结束
		}
		line = strings.TrimSpace(line) // 去除首尾空白
		if line == "" {
			break // 空行结束头部
		}
		sep := strings.Index(line, ":") // 查找冒号分隔符
		if sep < 0 {
			continue // 没有冒号跳过
		}
		key := strings.TrimSpace(line[:sep]) // 获取头部键
		value := strings.TrimSpace(line[sep+1:]) // 获取头部值
		headers[key] = value // 存入map
	}

	body, _ := io.ReadAll(reader) // 读取剩余内容为请求体

	url := path // 默认url为path
	if !strings.HasPrefix(path, "http") { // 如果path不是http开头
		host, ok := headers["Host"] // 获取Host头
		if !ok {
			return nil, fmt.Errorf("缺少 Host 头") // 没有Host头报错
		}
		scheme := "https" // 默认https
		if strings.HasPrefix(path, "http://") { // 如果path是http://开头
			scheme = "http" // 切换为http
		}
		url = fmt.Sprintf("%s://%s%s", scheme, host, path) // 拼接完整url
	}

	return &RequestTemplate{
		Method:  method, // 方法
		URL:     url,    // url
		Headers: headers, // 头部
		Body:    body,    // 请求体
	}, nil
}

// 加载所有请求模板
func loadTemplates() ([]*RequestTemplate, error) {
	files, err := filepath.Glob("Post/post*.txt") // 查找所有Post/post*.txt文件
	if err != nil {
		return nil, err // 查找失败返回错误
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("在Post目录下未找到post*.txt文件") // 没有模板文件报错
	}
	var templates []*RequestTemplate // 模板切片
	for _, file := range files {
		data, err := os.ReadFile(file) // 读取文件内容
		if err != nil {
			return nil, fmt.Errorf("读取模板文件 %s 失败: %w", file, err) // 读取失败报错
		}
		template, err := parseRawRequest(data) // 解析模板
		if err != nil {
			return nil, fmt.Errorf("解析模板 %s 失败: %w", file, err) // 解析失败报错
		}
		template.Name = strings.TrimSuffix(filepath.Base(file), ".txt") // 设置模板名称
		log.SetFlags(0) 
		log.Printf("[%s]hahahhaha ", template.Name) 
		templates = append(templates, template) // 加入切片
	}
	return templates, nil // 返回模板切片
}

// 发送HTTP请求并打印响应
func sendRequest(template *RequestTemplate) {
	
	client := &http.Client{} // 创建HTTP客户端
	req, err := http.NewRequest(template.Method, template.URL, bytes.NewReader(template.Body)) // 创建请求
	if err != nil {
		log.Printf("[%s] 创建请求失败: %v", template.Name, err) // 创建失败打印日志
		return
	}
	for k, v := range template.Headers { // 设置请求头
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req) // 发送请求
	if err != nil {
		log.Printf("[%s] 请求失败: %v", template.Name, err) // 请求失败打印日志
		return
	}
	defer resp.Body.Close() // 延迟关闭响应体
	body, err := io.ReadAll(resp.Body) // 读取响应体
	if err != nil {
		log.Printf("[%s] 读取响应失败: %v", template.Name, err) // 读取失败打印日志
		return
	}
	log.Printf("[%s] 状态码: %d | 响应: %s", template.Name, resp.StatusCode, strings.TrimSpace(string(body))) // 打印状态码和响应内容
}

//Go 程序必须有 main 包和 main() 函数作为入口。
func main() {
	log.SetFlags(0) 
	templates, err := loadTemplates() // 加载请求模板
	if err != nil {
		log.Fatalf("加载请求模板失败: %v", err) // 加载失败终止程序
	}
	log.Printf("已加载 %d 个请求模板，使用 %d 个线程并发...", len(templates), threads) // 打印模板数量和线程数

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM) // 创建带信号通知的上下文
	defer stop() // 程序结束时取消上下文
	//Go 原生支持并发，推荐用 goroutine + channel + sync 包实现高效并发。
	var wg sync.WaitGroup // 创建WaitGroup用于等待所有请求完成
	sem := make(chan struct{}, threads) // 创建信号量控制并发数

loop:
	for {
		select {
		case <-ctx.Done(): // 检查是否收到中断信号
			log.Println("检测到中断信号，准备退出...") // 打印退出信息
			break loop // 跳出循环
		default:
			for _, template := range templates { // 遍历所有模板
				wg.Add(1) // 增加WaitGroup计数
				sem <- struct{}{} // 占用一个信号量
				go func(t *RequestTemplate) { // 启动goroutine发送请求
					defer wg.Done() // 请求完成减少WaitGroup计数
					defer func() { <-sem }() // 释放信号量
					sendRequest(t) // 发送请求
				}(template)
			}
			time.Sleep(time.Duration(interval) * time.Second) // 每轮请求后等待指定间隔
		}
	}
	wg.Wait() // 等待所有请求完成
	log.Println("所有请求已完成，程序退出。") // 打印程序结束信息
}