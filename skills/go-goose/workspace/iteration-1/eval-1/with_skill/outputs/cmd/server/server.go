package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	user "github.com/soyacen/goose/skills/go-goose/workspace/iteration-1/eval-1/with_skill/outputs"
)

type userService struct {
	mu    sync.RWMutex
	users map[int64]*user.User
	nextID int64
}

func newUserService() *userService {
	return &userService{
		users:  make(map[int64]*user.User),
		nextID: 1,
	}
}

func (s *userService) CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.CreateUserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newUser := &user.User{
		Id:    s.nextID,
		Name:  req.Name,
		Email: req.Email,
	}
	s.users[s.nextID] = newUser
	s.nextID++

	return &user.CreateUserResponse{
		User: newUser,
	}, nil
}

func (s *userService) GetUser(ctx context.Context, req *user.GetUserRequest) (*user.GetUserResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[req.Id]
	if !ok {
		return nil, fmt.Errorf("user with id %d not found", req.Id)
	}

	return &user.GetUserResponse{
		User: u,
	}, nil
}

func (s *userService) DeleteUser(ctx context.Context, req *user.DeleteUserRequest) (*user.DeleteUserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.users[req.Id]
	if !ok {
		return nil, fmt.Errorf("user with id %d not found", req.Id)
	}

	delete(s.users, req.Id)

	return &user.DeleteUserResponse{
		Success: true,
	}, nil
}

func main() {
	router := http.NewServeMux()
	service := newUserService()
	router = user.AppendUserServiceHttpRoute(router, service)

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
