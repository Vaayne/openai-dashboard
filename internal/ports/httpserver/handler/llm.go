package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/Vaayne/aienvoy/internal/core/llmservice"
	"github.com/Vaayne/aienvoy/internal/pkg/config"
	"github.com/Vaayne/aienvoy/pkg/llm"
	"github.com/Vaayne/aienvoy/pkg/llm/openai"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/daos"
)

type LLMHandler struct{}

func NewLLMHandler() *LLMHandler {
	return &LLMHandler{}
}

type CreateConversationRequest struct {
	Name  string `json:"name,omitempty"`
	Model string `json:"model"`
}

func (l *LLMHandler) CreateConversation(c echo.Context) error {
	ctx := c.Request().Context()
	req := new(CreateConversationRequest)
	if err := c.Bind(req); err != nil {
		slog.ErrorContext(ctx, "bind create conversation request body error", "err", err.Error())
		return c.String(http.StatusBadRequest, "bad request")
	}
	svc := newLlmService(c, req.Model)

	cov, err := svc.CreateConversation(ctx, req.Name)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, cov)
}

func (l *LLMHandler) ListConversations(c echo.Context) error {
	ctx := c.Request().Context()
	svc := newLlmService(c, openai.ModelGPT3Dot5Turbo)
	covs, err := svc.ListConversations(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, covs)
}

func (l *LLMHandler) GetConversation(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.PathParam("id")
	svc := newLlmService(c, openai.ModelGPT3Dot5Turbo)
	cov, err := svc.GetConversation(ctx, id)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, cov)
}

func (l *LLMHandler) DeleteConversation(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.PathParam("id")
	svc := newLlmService(c, openai.ModelGPT3Dot5Turbo)
	err := svc.DeleteConversation(ctx, id)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, nil)
}

func (l *LLMHandler) CreateMessage(c echo.Context) error {
	ctx := c.Request().Context()
	conversationId := c.PathParam("conversationId")
	req := new(llm.ChatCompletionRequest)
	err := c.Bind(req)
	if err != nil {
		slog.ErrorContext(ctx, "bind chat request body error", "err", err.Error())
		return c.String(http.StatusBadRequest, "bad request")
	}

	if req.Stream {
		return l.createMessageStream(c, conversationId, req)
	}

	svc := newLlmService(c, req.Model)
	msg, err := svc.CreateMessage(ctx, conversationId, *req)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, msg)
}

func (l *LLMHandler) createMessageStream(c echo.Context, conversationId string, req *llm.ChatCompletionRequest) error {
	svc := newLlmService(c, req.Model)
	dataChan := make(chan llm.ChatCompletionStreamResponse)
	defer close(dataChan)
	errChan := make(chan error)
	defer close(errChan)

	go svc.CreateMessageStream(c.Request().Context(), conversationId, *req, dataChan, errChan)

	// sse stream response
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	c.Response().WriteHeader(http.StatusOK)

	for {
		select {
		case data := <-dataChan:
			msg, err := json.Marshal(data)
			if err != nil {
				slog.ErrorContext(c.Request().Context(), "chat stream marshal response error", "err", err.Error())
				return c.String(http.StatusInternalServerError, err.Error())
			}
			_, err = c.Response().Write([]byte(fmt.Sprintf("data: %s\n\n", msg)))
			if err != nil {
				slog.ErrorContext(c.Request().Context(), "write chat stream response error", "err", err.Error())
				return c.String(http.StatusInternalServerError, err.Error())
			}
			c.Response().Flush()
		case err := <-errChan:
			if errors.Is(err, io.EOF) {
				return c.String(http.StatusOK, "data: [DONE]\n\n")
			}
			return c.String(http.StatusInternalServerError, err.Error())
		}
	}
}

func (l *LLMHandler) ListMessages(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.PathParam("conversationId")
	svc := newLlmService(c, openai.ModelGPT3Dot5Turbo)
	msgs, err := svc.ListMessages(ctx, id)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, msgs)
}

func (l *LLMHandler) GetMessage(c echo.Context) error {
	ctx := c.Request().Context()
	// conversationId := c.PathParam("conversationId")
	messageId := c.PathParam("messageId")
	svc := newLlmService(c, openai.ModelGPT3Dot5Turbo)
	msg, err := svc.GetMessage(ctx, messageId)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, msg)
}

func (l *LLMHandler) DeleteMessage(c echo.Context) error {
	ctx := c.Request().Context()
	// conversationId := c.PathParam("conversationId")
	messageId := c.PathParam("messageId")
	svc := newLlmService(c, openai.ModelGPT3Dot5Turbo)
	err := svc.DeleteMessage(ctx, messageId)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, nil)
}

func (l *LLMHandler) CreateChatCompletion(c echo.Context) error {
	ctx := c.Request().Context()
	req := new(llm.ChatCompletionRequest)
	err := c.Bind(req)
	if err != nil {
		slog.ErrorContext(ctx, "bind chat request body error", "err", err.Error())
		return c.String(http.StatusBadRequest, "bad request")
	}

	svc := newLlmService(c, req.Model)
	if svc == nil {
		return c.String(http.StatusBadRequest, "unknown model")
	}

	if req.Stream {
		return l.chatStream(c, svc, *req)
	}

	resp, err := svc.CreateChatCompletion(ctx, *req)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, resp)
}

func (l *LLMHandler) chatStream(c echo.Context, svc llmservice.Service, req llm.ChatCompletionRequest) error {
	dataChan := make(chan llm.ChatCompletionStreamResponse)
	defer close(dataChan)
	errChan := make(chan error)
	defer close(errChan)

	go svc.CreateChatCompletionStream(c.Request().Context(), req, dataChan, errChan)

	// sse stream response
	c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	c.Response().WriteHeader(http.StatusOK)

	for {
		select {
		case data := <-dataChan:
			msg, err := json.Marshal(data)
			if err != nil {
				slog.ErrorContext(c.Request().Context(), "chat stream marshal response error", "err", err.Error())
				return c.String(http.StatusInternalServerError, err.Error())
			}
			_, err = c.Response().Write([]byte(fmt.Sprintf("data: %s\n\n", msg)))
			if err != nil {
				slog.ErrorContext(c.Request().Context(), "write chat stream response error", "err", err.Error())
				return c.String(http.StatusInternalServerError, err.Error())
			}
			c.Response().Flush()
		case err := <-errChan:
			if errors.Is(err, io.EOF) {
				return c.String(http.StatusOK, "data: [DONE]\n\n")
			}
			return c.String(http.StatusInternalServerError, err.Error())
		}
	}
}

func newLlmService(c echo.Context, model string) llmservice.Service {
	return llmservice.New(model, llmservice.NewDao(c.Get(config.ContextKeyDao).(*daos.Dao)))
}
