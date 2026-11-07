package main

import (
	"context"
	"errors"
	"net"
)

func updateCheckMessage(err error) string {
	if err == nil {
		return "已经是最新版本"
	}
	// MyGo 0.7.4 keeps ErrNotPublished internal. Match only its wrapped
	// sentinel, not arbitrary HTTP errors or a missing update manifest.
	for cause := err; cause != nil; cause = errors.Unwrap(cause) {
		if cause.Error() == "no release is published" {
			return "暂无可用更新"
		}
	}
	if errors.Is(err, context.Canceled) {
		return "已取消检查更新"
	}
	var networkError net.Error
	if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &networkError) && networkError.Timeout() {
		return "检查更新超时，请稍后重试。"
	}
	return "暂时无法检查更新，请稍后重试。"
}
