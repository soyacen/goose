package main

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/soyacen/goose/skills/go-goose/workspace/iteration-1/eval-1/without_skill/outputs/genproto/v1"
)

type userServer struct {
	v1.UnimplementedUserServiceServer
	users map[string]*v1.User
}

func NewUserServer() *userServer {
	return &userServer{
		users: make(map[string]*v1.User),
	}
}

func (s *userServer) CreateUser(ctx context.Context, req *v1.CreateUserRequest) (*v1.CreateUserResponse, error) {
	if req.Name == "" || req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "name and email are required")
	}

	id := fmt.Sprintf("user-%d", len(s.users)+1)
	user := &v1.User{
		Id:    id,
		Name:  req.Name,
		Email: req.Email,
	}
	s.users[id] = user

	return &v1.CreateUserResponse{User: user}, nil
}

func (s *userServer) GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.GetUserResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	user, exists := s.users[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &v1.GetUserResponse{User: user}, nil
}

func (s *userServer) DeleteUser(ctx context.Context, req *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	_, exists := s.users[req.Id]
	if !exists {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	delete(s.users, req.Id)

	return &v1.DeleteUserResponse{Success: true}, nil
}

func main() {
	// Create gRPC server
	grpcServer := grpc.NewServer()
	userServer := NewUserServer()
	v1.RegisterUserServiceServer(grpcServer, userServer)

	// Start gRPC server
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}

	fmt.Println("gRPC server starting on :50051")
	if err := grpcServer.Serve(listener); err != nil {
		fmt.Printf("Failed to serve: %v\n", err)
	}
}

// HTTP handlers for REST gateway (optional, using grpc-gateway would be better for production)
func httpHandler(userServer *userServer) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			// Create user handler would go here
			w.WriteHeader(http.StatusNotImplemented)
			w.Write([]byte(`{"error": "Use gRPC client to create users"}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/v1/users/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/v1/users/"):]
		switch r.Method {
		case http.MethodGet:
			// Get user handler would go here
			w.WriteHeader(http.StatusNotImplemented)
			w.Write([]byte(`{"error": "Use gRPC client to get users"}`))
		case http.MethodDelete:
			// Delete user handler would go here
			w.WriteHeader(http.StatusNotImplemented)
			w.Write([]byte(`{"error": "Use gRPC client to delete users"}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		_ = id
	})

	return mux
}
