// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package assistant

import (
	"context"
	"sync/atomic"
)

type Executor interface {
	Run(ctx context.Context, assistant *Assistant, thread *Thread, message Message, opts []Option) error
	ShutdownAssistant(ctx context.Context, assistant *Assistant) error
}

func SetDefaultExecutor(executor Executor) {
	if executor != nil {
		defaultExecutor.Store(&executor)
	}
}

var defaultExecutor atomic.Pointer[Executor] //nolint:gochecknoglobals
