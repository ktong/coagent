// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package internal

import (
	"testing"

	"github.com/ktong/assistant/internal/assert"
)

func TestConst(t *testing.T) {
	assert.Equal[string](t, "__sub_thread_ids__", keySubThreadIDs)
}
