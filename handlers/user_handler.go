package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"go-microservice/models"
	"go-microservice/services"
)

type UserHandler struct {
	userService  *services.UserService
	auditService *services.AuditService
}

func NewUserHandler(userService *services.UserService, auditService *services.AuditService) *UserHandler {
	return &UserHandler{
		userService:  userService,
		auditService: auditService,
	}
}

func (h *UserHandler) Register(r *mux.Router) {
	r.HandleFunc("/users", h.ListUsers).Methods(http.MethodGet)
	r.HandleFunc("/users/{id}", h.GetUser).Methods(http.MethodGet)
	r.HandleFunc("/users", h.CreateUser).Methods(http.MethodPost)
	r.HandleFunc("/users/{id}", h.UpdateUser).Methods(http.MethodPut)
	r.HandleFunc("/users/{id}", h.DeleteUser).Methods(http.MethodDelete)
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.userService.List())
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := h.userService.Get(id)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	savedUser, err := h.userService.Create(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	go h.auditService.LogUserAction("CREATE", savedUser.ID, r.RemoteAddr)
	go h.auditService.SendNotification("user_created", savedUser.Email)

	writeJSON(w, http.StatusCreated, savedUser)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	updatedUser, err := h.userService.Update(id, user)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	go h.auditService.LogUserAction("UPDATE", updatedUser.ID, r.RemoteAddr)
	go h.auditService.SendNotification("user_updated", updatedUser.Email)

	writeJSON(w, http.StatusOK, updatedUser)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.userService.Delete(id); err != nil {
		if errors.Is(err, services.ErrNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go h.auditService.LogUserAction("DELETE", id, r.RemoteAddr)
	go h.auditService.SendNotification("user_deleted", strconv.Itoa(id))

	w.WriteHeader(http.StatusNoContent)
}

func pathID(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
