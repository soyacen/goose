package main

import (
	"context"
	"fmt"
	"log"

	"github.com/soyacen/goose/example/pagination/v1"
)

func main() {
	// Create HTTP client pointing to the server
	// Generated constructor: New{ServiceName}HttpClient
	client := pagination.NewPaginationHttpClient("http://localhost:8080")

	ctx := context.Background()

	// Example 1: Basic pagination request
	// The client will call: GET /v1/items?page_num=1&page_size=10
	resp, err := client.ListItems(ctx, &pagination.ListItemsRequest{
		PageNum:  1,
		PageSize: 10,
	})
	if err != nil {
		log.Fatalf("Failed to list items: %v", err)
	}

	fmt.Printf("Page %d of %d (total: %d items)\n",
		resp.Meta.PageNum,
		resp.Meta.TotalPages,
		resp.Meta.TotalCount)

	for _, item := range resp.Items {
		fmt.Printf("  - Item %d: %s (%s)\n", item.Id, item.Name, item.Status)
	}

	// Example 2: Pagination with filters
	// The client will call: GET /v1/items?page_num=2&page_size=20&status=active&sort_by=name
	respFiltered, err := client.ListItems(ctx, &pagination.ListItemsRequest{
		PageNum:  2,
		PageSize: 20,
		Status:   "active",
		SortBy:   "name",
	})
	if err != nil {
		log.Fatalf("Failed to list filtered items: %v", err)
	}

	fmt.Printf("\nFiltered results (page %d):\n", respFiltered.Meta.PageNum)
	for _, item := range respFiltered.Items {
		fmt.Printf("  - Item %d: %s\n", item.Id, item.Name)
	}

	// Example 3: Check pagination metadata
	if resp.Meta.HasNext {
		fmt.Printf("\nThere are more pages after page %d\n", resp.Meta.PageNum)
	}
	if resp.Meta.HasPrev {
		fmt.Printf("There are previous pages before page %d\n", resp.Meta.PageNum)
	}
}
