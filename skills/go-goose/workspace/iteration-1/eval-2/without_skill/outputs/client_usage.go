package main

import (
	"context"
	"fmt"
	"log"
)

// This example demonstrates how a client calls the pagination API
// with query parameters page_num and page_size.

func main() {
	// Initialize the HTTP client
	// The generated client will automatically serialize the request fields
	// as query parameters for GET requests

	// Example 1: Basic pagination call
	{
		client := NewPaginationServiceHttpClient("http://localhost:8080")

		req := &ListItemsRequest{
			PageNum:  1,
			PageSize: 10,
		}

		resp, err := client.ListItems(context.Background(), req)
		if err != nil {
			log.Fatalf("ListItems failed: %v", err)
		}

		fmt.Printf("Page %d, Size %d, Total %d\n",
			resp.GetPageNum(),
			resp.GetPageSize(),
			resp.GetTotal())

		for _, item := range resp.GetList() {
			fmt.Printf("  - Item: %d, %s\n", item.GetId(), item.GetName())
		}
	}

	// Example 2: Pagination with additional query parameters
	{
		client := NewPaginationServiceHttpClient("http://localhost:8080")

		req := &ListUsersRequest{
			PageNum:  2,
			PageSize: 20,
			Keyword:  "john",
		}

		resp, err := client.ListUsers(context.Background(), req)
		if err != nil {
			log.Fatalf("ListUsers failed: %v", err)
		}

		fmt.Printf("Page %d, Size %d, Total %d\n",
			resp.GetPageNum(),
			resp.GetPageSize(),
			resp.GetTotal())

		for _, user := range resp.GetList() {
			fmt.Printf("  - User: %d, %s, %s\n",
				user.GetId(),
				user.GetName(),
				user.GetEmail())
		}
	}

	// Example 3: Making the HTTP request directly (without generated client)
	{
		// You can also make the HTTP request directly using any HTTP client
		// The query parameters will be:
		//   page_num=1&page_size=10

		// Using standard library
		// req, err := http.NewRequest("GET",
		// 	"http://localhost:8080/v1/items?page_num=1&page_size=10",
		// 	nil)
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// resp, err := http.DefaultClient.Do(req)
		// ...
	}
}

// ---- Generated Client Interface ----
// The goose framework generates the following client interface:
//
// type PaginationServiceClient interface {
//     ListItems(ctx context.Context, req *ListItemsRequest) (*ListItemsResponse, error)
//     ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error)
// }
//
// The HTTP client implementation will automatically serialize the request
// fields as query parameters for GET endpoints.
//
// HTTP Request Examples:
//
//   GET /v1/items?page_num=1&page_size=10
//   GET /v1/users?page_num=2&page_size=20&keyword=john
