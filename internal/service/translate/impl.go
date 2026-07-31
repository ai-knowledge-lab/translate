package translate

import (
	"time"

	"github.com/go-resty/resty/v2"
)

type Service struct {
}

func NewService() *Service {
	return &Service{}
}

func duration() time.Duration {
	return time.Second * 3600
}

var (
	client = resty.New().
		SetTimeout(duration()).
		SetHeader("Content-Type", "application/json")
)
