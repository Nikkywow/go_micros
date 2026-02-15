package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type auditEvent struct {
	Action    string    `json:"action"`
	UserID    int       `json:"user_id"`
	RemoteIP  string    `json:"remote_ip"`
	CreatedAt time.Time `json:"created_at"`
}

type notifyEvent struct {
	Topic     string    `json:"topic"`
	Recipient string    `json:"recipient"`
	CreatedAt time.Time `json:"created_at"`
}

type AuditService struct {
	logger      *slog.Logger
	integration *IntegrationService

	auditCh  chan auditEvent
	notifyCh chan notifyEvent
	errCh    chan error

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewAuditService(logger *slog.Logger, integration *IntegrationService) *AuditService {
	ctx, cancel := context.WithCancel(context.Background())
	s := &AuditService{
		logger:      logger,
		integration: integration,
		auditCh:     make(chan auditEvent, 2048),
		notifyCh:    make(chan notifyEvent, 2048),
		errCh:       make(chan error, 2048),
		ctx:         ctx,
		cancel:      cancel,
	}
	s.start()
	return s
}

func (s *AuditService) LogUserAction(action string, userID int, remoteIP string) {
	select {
	case s.auditCh <- auditEvent{
		Action:    action,
		UserID:    userID,
		RemoteIP:  remoteIP,
		CreatedAt: time.Now().UTC(),
		}:
	default:
		s.pushErr(fmt.Errorf("audit queue full"))
	}
}

func (s *AuditService) SendNotification(topic, recipient string) {
	select {
	case s.notifyCh <- notifyEvent{
		Topic:     topic,
		Recipient: recipient,
		CreatedAt: time.Now().UTC(),
		}:
	default:
		s.pushErr(fmt.Errorf("notify queue full"))
	}
}

func (s *AuditService) Close() {
	s.cancel()
	s.wg.Wait()
}

func (s *AuditService) start() {
	s.wg.Add(3)

	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ctx.Done():
				return
			case ev := <-s.auditCh:
				s.handleAudit(ev)
			}
		}
	}()

	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ctx.Done():
				return
			case ev := <-s.notifyCh:
				s.logger.Info("notification", "topic", ev.Topic, "recipient", ev.Recipient)
			}
		}
	}()

	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ctx.Done():
				return
			case err := <-s.errCh:
				s.logger.Error("async worker error", "error", err)
			}
		}
	}()
}

func (s *AuditService) handleAudit(ev auditEvent) {
	s.logger.Info("audit", "action", ev.Action, "user_id", ev.UserID, "remote_ip", ev.RemoteIP)

	if !s.integration.Enabled() {
		return
	}

	payload, err := json.Marshal(ev)
	if err != nil {
		s.pushErr(fmt.Errorf("marshal audit event: %w", err))
		return
	}

	name := fmt.Sprintf("audit/%s-user-%d.json", ev.CreatedAt.Format(time.RFC3339Nano), ev.UserID)
	if err := s.integration.UploadAuditLog(s.ctx, name, payload); err != nil {
		s.pushErr(err)
	}
}

func (s *AuditService) pushErr(err error) {
	select {
	case s.errCh <- err:
	default:
		s.logger.Error("error queue full", "error", err)
	}
}
