package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// The ways an answer can go wrong. Each of them is a sentence a person can
// act on: what the HTTP library said is not one of them.
var (
	ErrNotConfigured = errors.New("还没有配置 AI：请到设置 · AI 里启用并选择模型")
	ErrNoModel       = errors.New("没有选中模型：请到设置 · AI 里选择一个模型")
	ErrAuth          = errors.New("认证失败：检查密钥是否正确")
	ErrRateLimit     = errors.New("被服务商限流了，请稍后重试")
	ErrTimeout       = errors.New("等待响应超时：可以重试，或在设置里调大超时")
	// ErrTruncated is a stream that went dead before it said it was
	// over. It is not the same as a model that ran out of budget: that
	// one came to an end of its own, this one did not.
	ErrTruncated   = errors.New("连接中断，结果可能不完整")
	ErrEmptyResult = errors.New("模型没有返回正文（可能是思考占用了全部预算）" +
		"：请重试或关闭「思考」")
	ErrCanceled = errors.New("已取消")
	// ErrSilent is a model that was asked and said nothing at all, not
	// even a first word, before it was given up on.
	ErrSilent = errors.New("the model said nothing")
)

// Network is a model that could not be reached at all: the sentence says
// what a person can do about it.
func Network(detail string) error {
	return unreachable{detail: detail}
}

// Protocol is a model that answered something that could not be read.
func Protocol(detail string) error {
	return fmt.Errorf("返回内容无法解析（%s）", detail)
}

// NoSuchModel is a model the server has never heard of.
func NoSuchModel(name string) error {
	return fmt.Errorf("模型不存在：%s", name)
}

// unreachable is a model that could not be reached.
type unreachable struct{ detail string }

// Error says what happened, and what to do about it.
func (e unreachable) Error() string {
	return fmt.Sprintf("网络不可用（%s）：检查地址是否正确、服务是否已启动",
		e.detail)
}

// Status is an answer that came back with a status that is not a good one.
type Status struct {
	Code int
	Body string
}

// Error says the status, and whatever came with it.
func (s Status) Error() string {
	return fmt.Sprintf("the model answered %d: %s", s.Code, s.Body)
}

// failed is what a status means to the one who was waiting for words.
func (s Status) failed(model string) error {
	switch {
	case s.Code == http.StatusUnauthorized, s.Code == http.StatusForbidden:
		return ErrAuth
	case s.Code == http.StatusTooManyRequests:
		return ErrRateLimit
	case s.Code == http.StatusNotFound:
		return NoSuchModel(model)
	}
	return Protocol(s.Error())
}

// Retryable is whether asking again is worth the words. A key that is
// wrong, or a model that does not exist, comes back the same however
// often it is asked; a model that was busy, or slow, or that broke off
// mid-sentence, may well answer whole the second time.
func Retryable(err error) bool {
	if errors.Is(err, ErrRateLimit) || errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrTruncated) {
		return true
	}
	var never unreachable
	return errors.As(err, &never)
}

// failure is what went wrong, in words a person can act on, told which
// model it was that failed so the sentence can name it.
func failure(err error, model string) error {
	switch {
	case errors.Is(err, context.Canceled):
		return ErrCanceled
	case errors.Is(err, ErrSilent), errors.Is(err, context.DeadlineExceeded):
		return ErrTimeout
	case errors.Is(err, ErrTruncated):
		return ErrTruncated
	}
	var timed interface{ Timeout() bool }
	if errors.As(err, &timed) && timed.Timeout() {
		return ErrTimeout
	}
	var status Status
	if errors.As(err, &status) {
		return status.failed(model)
	}
	return Network(err.Error())
}
