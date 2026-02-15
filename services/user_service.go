package services

import (
	"errors"
	"sort"
	"sync"

	"go-microservice/models"
)

var ErrNotFound = errors.New("user not found")

type UserService struct {
	mu     sync.RWMutex
	users  map[int]models.User
	nextID int
}

func NewUserService() *UserService {
	return &UserService{
		users:  make(map[int]models.User),
		nextID: 1,
	}
}

func (s *UserService) List() []models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]models.User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, u)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func (s *UserService) Get(id int) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.users[id]
	if !ok {
		return models.User{}, ErrNotFound
	}
	return u, nil
}

func (s *UserService) Create(u models.User) (models.User, error) {
	if err := u.Validate(); err != nil {
		return models.User{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	u.ID = s.nextID
	s.nextID++
	s.users[u.ID] = u
	return u, nil
}

func (s *UserService) Update(id int, u models.User) (models.User, error) {
	if err := u.Validate(); err != nil {
		return models.User{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[id]; !ok {
		return models.User{}, ErrNotFound
	}
	u.ID = id
	s.users[id] = u
	return u, nil
}

func (s *UserService) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[id]; !ok {
		return ErrNotFound
	}
	delete(s.users, id)
	return nil
}
