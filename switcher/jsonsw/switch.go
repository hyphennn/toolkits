// Package jsonsw
// Create-time: 2024/11/22
package jsonsw

import (
	"github.com/bytedance/sonic"
	"github.com/hyphennn/glambda/gvalue"
)

func Switch[F, T any](f F) (T, error) {
	t := gvalue.Zero[T]()
	tmp, err := sonic.Marshal(f)
	if err != nil {
		return t, err
	}
	err = sonic.Unmarshal(tmp, &t)
	return t, err
}
