package v1

import (
	chatpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/chat/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/services"
)

type ChatHandler struct {
	chatpb.UnimplementedChatServiceServer
	chatService *services.ChatService
	logger      logging.Logger
}

func NewChatHandler(
	chatService *services.ChatService,
	logger logging.Logger,
) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		logger:      logger,
	}
}
