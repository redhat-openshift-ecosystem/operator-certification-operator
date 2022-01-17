package reconcilers

import (
	"context"

	"github.com/go-logr/logr"
	imagev1 "github.com/openshift/api/image/v1"
	certv1alpha1 "github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	"github.com/redhat-openshift-ecosystem/operator-certification-operator/internal/objects"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	marketplaceIndex = "redhat-marketplace-index"
)

type MarketplaceImageStreamReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func NewMarketplaceImageStreamReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *MarketplaceImageStreamReconciler {
	return &MarketplaceImageStreamReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

// reconcileMarketplaceImageStream will ensure that the Red Hat Marketplace ImageStream is present and up to date.
func (r *MarketplaceImageStreamReconciler) Reconcile(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline) (bool, error) {
	key := types.NamespacedName{
		Namespace: pipeline.Namespace,
		Name:      marketplaceIndex,
	}
	log := r.Log.WithValues("marketplaceimagestream", key)

	stream := newImageStream(key)
	if objects.IsObjectFound(ctx, r.Client, key, stream) {
		log.Info("existing marketplace image stream found")
		return false, nil // Existing ImageStream found, do nothing...
	}

	imgImport := newImageStreamImport(key)
	imgImport.Spec.Import = true
	imgImport.Spec.Repository = &imagev1.RepositoryImportSpec{
		From: corev1.ObjectReference{
			Kind: "DockerImage",
			Name: "registry.redhat.io/redhat/redhat-marketplace-index",
		},
		ImportPolicy: imagev1.TagImportPolicy{
			Scheduled: true,
		},
		ReferencePolicy: imagev1.TagReferencePolicy{
			Type: imagev1.LocalTagReferencePolicy,
		},
	}

	log.Info("creating new marketplace image stream import")
	if err := r.Client.Create(ctx, imgImport); err != nil {
		return true, err
	}

	return false, nil
}
