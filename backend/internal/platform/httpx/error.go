package httpx

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
)

type AppError struct {
	Status  int
	Code    string
	Message string
	Err     error
}

func NewError(status int, code, message string, err error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Err: err}
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func ErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		appError := &AppError{
			Status:  fiber.StatusInternalServerError,
			Code:    "INTERNAL_ERROR",
			Message: "an unexpected error occurred",
			Err:     err,
		}

		var knownError *AppError
		if errors.As(err, &knownError) {
			appError = knownError
		} else {
			var fiberError *fiber.Error
			if errors.As(err, &fiberError) {
				appError.Status = fiberError.Code
				appError.Code = "HTTP_ERROR"
				appError.Message = fiberError.Message
			}
		}

		if appError.Status >= fiber.StatusInternalServerError {
			logger.Error("request failed", "method", c.Method(), "path", c.Path(), "error", err)
		}

		return c.Status(appError.Status).JSON(Failure(appError.Code, appError.Message))
	}
}
