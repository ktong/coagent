// Copyright (c) 2024 the authors
// Use of this source code is governed by a MIT license found in the LICENSE file.

package embedded

type Tool interface {
	tool()
}

type BuiltInTool interface {
	Tool

	builtInTool()
}
