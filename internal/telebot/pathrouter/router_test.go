package pathrouter

import (
	"context"
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestPathParser(t *testing.T) {
	tests := []struct {
		url           string
		expectedError error
	}{
		{
			url:           "/products/12345",
			expectedError: nil,
		},
		{
			url:           "/articles/technology/",
			expectedError: nil,
		},
		{
			url:           "/articles/science/42",
			expectedError: nil,
		},
		{
			url:           "/unknown/path",
			expectedError: ErrNotFoundHandler,
		},
	}

	parser := NewRouter()
	parser.HandleFunc("/products/{key}", func(
		ctx context.Context,
		vars map[string]string,
		userId int64,
		update tgbotapi.Update,
	) error {
		_, ok := vars["key"]
		if !ok {
			return errors.New("missing vars")
		}
		return nil
	})

	parser.HandleFunc("/articles/{category}/", func(
		ctx context.Context,
		vars map[string]string,
		userId int64,
		update tgbotapi.Update,
	) error {
		_, ok := vars["category"]
		if !ok {
			return errors.New("missing vars")
		}
		return nil
	})

	parser.HandleFunc("/articles/{category}/{id:[0-9]+}", func(
		ctx context.Context,
		vars map[string]string,
		userId int64,
		update tgbotapi.Update,
	) error {
		_, ok := vars["id"]
		if !ok {
			return errors.New("missing vars")
		}
		_, ok = vars["category"]
		if !ok {
			return errors.New("missing vars")
		}
		return nil
	})

	for _, tt := range tests {
		err := parser.Execute(context.TODO(), tt.url, 0, tgbotapi.Update{})
		if err != tt.expectedError {
			t.Errorf("For path '%s': expected error %v, got %v", tt.url, tt.expectedError, err)
		}
	}
}
