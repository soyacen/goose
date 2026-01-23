package outgoing_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/soyacen/goose/outgoing"
)

func Example_basicGet() {
	// 创建一个基本的GET请求
	result, err := outgoing.Get().
		URL(outgoing.URLString("https://httpbin.org/get")).
		Query(outgoing.SetQuery("key1", "value1"), outgoing.SetQuery("key2", "value2")).
		Header(outgoing.SetHeader("X-Custom-Header", "custom-value")).
		Body().
		Send(context.Background())
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", result.StatusCode())
	body, _ := result.TextBody()
	fmt.Printf("Response body: %s\n", body)
}

func Example_postJSON() {
	type Request struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	type Response struct {
		JSON map[string]interface{} `json:"json"`
	}

	req := Request{Name: "John Doe", Email: "john@example.com"}

	result, err := outgoing.Post().
		URL(outgoing.URLString("https://httpbin.org/post")).
		Query().
		Header(outgoing.SetHeader("Content-Type", "application/json")).
		Body(outgoing.JSONBody(req)).
		Send(context.Background())
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	var resp Response
	if err := result.JSONBody(&resp); err != nil {
		log.Printf("Error parsing response: %v", err)
		return
	}

	fmt.Printf("Posted name: %s\n", resp.JSON["json"].(map[string]interface{})["name"])
}

func Example_withClientAndTimeout() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	type Request struct {
		Message string `json:"message"`
	}

	req := Request{Message: "Hello World"}

	result, err := outgoing.Post(outgoing.Client(client)).
		URL(outgoing.URLString("https://httpbin.org/post")).
		Query().
		Header().
		Body(outgoing.JSONBody(req)).
		Send(context.Background())
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", result.StatusCode())
}

func Example_formPost() {
	formData := url.Values{}
	formData.Set("username", "testuser")
	formData.Set("password", "password123")

	result, err := outgoing.Post().
		URL(outgoing.URLString("https://httpbin.org/post")).
		Query().
		Header().
		Body(outgoing.FormBody(formData)).
		Send(context.Background())
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", result.StatusCode())
}

func Example_multipartUpload() {
	// 创建一个模拟的文件
	fileContent := strings.NewReader("this is a file content")
	var formData []*outgoing.FormData
	formData = append(formData, &outgoing.FormData{
		FieldName: "file",
		File:      fileContent,
		Filename:  "test.txt",
	})
	formData = append(formData, &outgoing.FormData{
		FieldName: "description",
		Value:     "Test file upload",
	})

	result, err := outgoing.Post().
		URL(outgoing.URLString("https://httpbin.org/post")).
		Query().
		Header().
		Body(outgoing.MultipartBody(formData...)).
		Send(context.Background())
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", result.StatusCode())
}

func Example_withMiddleware() {
	// 定义一个简单的中间件记录请求时间
	loggingMiddleware := func(ctx context.Context, req *http.Request, cli *http.Client, invoker outgoing.Invoker) (*http.Response, error) {
		start := time.Now()
		fmt.Printf("Making request to: %s %s\n", req.Method, req.URL.String())

		resp, err := invoker(ctx, req, cli)

		fmt.Printf("Request completed in: %v\n", time.Since(start))
		return resp, err
	}

	result, err := outgoing.Get(outgoing.Middlewares(loggingMiddleware)).
		URL(outgoing.URLString("https://httpbin.org/get")).
		Query().
		Header().
		Body().
		Send(context.Background())
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", result.StatusCode())
}

func Example_authentication() {
	result, err := outgoing.Get().
		URL(outgoing.URLString("https://httpbin.org/headers")).Query().
		Header(outgoing.BearerAuth("your-token-here")).Body().
		// 或者使用基本认证
		// Header(outgoing.BasicAuth("username", "password")).
		Send(context.Background())
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", result.StatusCode())
	body, _ := result.TextBody()
	fmt.Printf("Response: %s\n", body)
}

func Example_complexQuery() {
	type Params struct {
		Limit  int    `url:"limit"`
		Offset int    `url:"offset"`
		Sort   string `url:"sort"`
		Filter string `url:"filter"`
	}

	params := Params{
		Limit:  10,
		Offset: 20,
		Sort:   "created_at",
		Filter: "active",
	}

	result, err := outgoing.Get().
		URL(outgoing.URLString("https://httpbin.org/get")).
		Query(outgoing.QueryObject(params)).Header().Body().
		Send(context.Background())
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", result.StatusCode())
}

func Example_bytesBody() {
	payload := []byte(`{"custom_format":true,"data":"some binary data..."}`)

	result, err := outgoing.Post().
		URL(outgoing.URLString("https://httpbin.org/post")).Query().
		Header(outgoing.SetHeader("Content-Type", "application/custom-type")).
		Body(outgoing.BytesBody(payload, "application/custom-type")).
		Send(context.Background())
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", result.StatusCode())
}

func Example_handlingResponseBody() {
	result, err := outgoing.Get().
		URL(outgoing.URLString("https://httpbin.org/json")).Query().Header().Body().
		Send(context.Background())
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	// 获取字节形式的响应体
	bytesBody, err := result.BytesBody()
	if err != nil {
		log.Printf("Error reading bytes: %v", err)
		return
	}
	fmt.Printf("Body size: %d bytes\n", len(bytesBody))

	// 获取文本形式的响应体
	textBody, err := result.TextBody()
	if err != nil {
		log.Printf("Error reading text: %v", err)
		return
	}
	fmt.Printf("First 50 chars of response: %.50s...\n", textBody)

	// 将响应写入自定义的writer
	var buf bytes.Buffer
	err = result.Body(&buf)
	if err != nil {
		log.Printf("Error writing to buffer: %v", err)
		return
	}
	fmt.Printf("Wrote %d bytes to buffer\n", buf.Len())
}

func Example_multipleHeadersAndCookies() {
	// 创建一个cookie
	cookie := &http.Cookie{
		Name:  "session_id",
		Value: "abc123",
	}

	result, err := outgoing.Put().
		URL(outgoing.URLString("https://httpbin.org/put")).Query().
		Header(
			outgoing.SetHeader("X-Custom-Header", "value1"),
			outgoing.AddHeader("X-Custom-Header", "value2"), // 添加另一个同名头部
			outgoing.UserAgent("MyApp/1.0"),
			outgoing.SetCookie(cookie),
		).
		Body(outgoing.TextBody(`{"updated": true}`, "application/json")).
		Send(context.Background())
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	fmt.Printf("Status: %d\n", result.StatusCode())
}
