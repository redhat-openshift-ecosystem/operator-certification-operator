package objects

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FetchObject will retrieve the object with the given namespace and name using the Kubernetes API.
// The result will be stored in the given object.
func fetchObject(ctx context.Context, client client.Client, key types.NamespacedName, obj client.Object) error {
	return client.Get(ctx, key, obj)
}

// IsObjectFound will perform a basic check that the given object exists via the Kubernetes API.
// If an error occurs as part of the check, the function will return false.
func IsObjectFound(ctx context.Context, client client.Client, key types.NamespacedName, obj client.Object) bool {
	return !apierrors.IsNotFound(fetchObject(ctx, client, key, obj))
}
