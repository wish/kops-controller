package fallbackidentity

import (
	"context"
	corev1 "k8s.io/api/core/v1"
)

type Identifier interface {
	IdentifyNode(ctx context.Context, node *corev1.Node) (map[string]string, error)
}
