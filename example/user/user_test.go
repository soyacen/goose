package user

import (
	"context"
	errors "errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// ---- Mock Service ----

type MockUserService struct{}

func (m *MockUserService) CreateUser(ctx context.Context, req *CreateUserRequest) (*CreateUserResponse, error) {
	return &CreateUserResponse{Item: &UserItem{Id: 1, Name: req.Name}}, nil
}

func (m *MockUserService) DeleteUser(ctx context.Context, req *DeleteUserRequest) (*DeleteUserResponse, error) {
	return &DeleteUserResponse{Id: req.GetId()}, nil
}

func (m *MockUserService) ModifyUser(ctx context.Context, req *ModifyUserRequest) (*ModifyUserResponse, error) {
	return &ModifyUserResponse{Id: req.GetId(), Name: req.GetName()}, nil
}

func (m *MockUserService) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UpdateUserResponse, error) {
	return &UpdateUserResponse{Id: req.GetId(), Item: &UserItem{Id: req.GetItem().GetId(), Name: req.GetItem().GetName()}}, nil
}

func (m *MockUserService) GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	return &GetUserResponse{Item: &UserItem{Id: req.Id, Name: "bob"}}, nil
}

func (m *MockUserService) ListUser(ctx context.Context, req *ListUserRequest) (*ListUserResponse, error) {
	return &ListUserResponse{
		PageNum:  req.GetPageNum(),
		PageSize: req.GetPageSize(),
		List: []*UserItem{
			{Id: 1, Name: "alice"},
			{Id: 3, Name: "bob"},
		},
	}, nil
}

func runServer(server *http.Server, port int) {
	router := http.NewServeMux()
	router = AppendUserHttpRoute(router, &MockUserService{})
	server.Addr = fmt.Sprintf(":%d", port)
	server.Handler = router
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

func newClient(port int) UserService {
	return NewUserHttpClient(fmt.Sprintf("http://localhost:%d", port))
}

// ---- Test Cases ----

func TestCreateUser(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 8081)
	time.Sleep(1 * time.Second)

	client := newClient(8081)
	resp, err := client.CreateUser(context.Background(), &CreateUserRequest{Name: "alice"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetItem().GetName() != "hello" && resp.GetItem().GetId() != 1 {
		t.Fatal("resp is not equal")
	}
}

func TestDeleteUser(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 8082)
	time.Sleep(1 * time.Second)

	client := newClient(8082)
	resp, err := client.DeleteUser(context.Background(), &DeleteUserRequest{Id: 1})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetId() != 1 {
		t.Fatal("resp is not equal")
	}
}

func TestModifyUser(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 8083)
	time.Sleep(1 * time.Second)

	client := newClient(8083)
	resp, err := client.ModifyUser(context.Background(), &ModifyUserRequest{Id: 2, Name: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetId() != 2 && resp.GetName() != "bob" {
		t.Fatal("resp is not equal")
	}
}

func TestUpdateUser(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 8084)
	time.Sleep(1 * time.Second)

	client := newClient(8084)
	resp, err := client.UpdateUser(context.Background(), &UpdateUserRequest{Id: 2, Item: &UserItem{Id: 3, Name: "bob"}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetId() != 2 && resp.GetItem().GetName() != "bob" && resp.GetItem().GetId() != 3 {
		t.Fatal("resp is not equal")
	}
}

func TestGetUser(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 8085)
	time.Sleep(1 * time.Second)

	client := newClient(8085)
	resp, err := client.GetUser(context.Background(), &GetUserRequest{Id: 3})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetItem().GetName() != "bob" && resp.GetItem().GetId() != 3 {
		t.Fatal("resp is not equal")
	}
}

func TestListUser(t *testing.T) {
	server := new(http.Server)
	defer server.Shutdown(context.Background())
	go runServer(server, 8086)
	time.Sleep(1 * time.Second)

	client := newClient(8086)
	resp, err := client.ListUser(context.Background(), &ListUserRequest{PageNum: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetPageNum() != 1 && resp.GetPageSize() != 10 &&
		resp.GetList()[0].GetId() != 1 && resp.GetList()[0].GetName() != "alice" &&
		resp.GetList()[1].GetId() != 2 && resp.GetList()[1].GetName() != "bob" {
		t.Fatal("resp is not equal")
	}
}
