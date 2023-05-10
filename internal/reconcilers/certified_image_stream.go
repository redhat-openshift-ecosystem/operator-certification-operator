package reconcilers

import (
	"context"

	"github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	"github.com/redhat-openshift-ecosystem/operator-certification-operator/internal/objects"

	"github.com/go-logr/logr"
	imagev1 "github.com/openshift/api/image/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	certifiedIndex = "certified-operator-index"
)

type CertifiedImageStreamReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func NewCertifiedImageStreamReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *CertifiedImageStreamReconciler {
	return &CertifiedImageStreamReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

// reconcileCertifiedImageStream will ensure that the certified operator ImageStream is present and up to date.
func (r *CertifiedImageStreamReconciler) Reconcile(ctx context.Context, pipeline *v1alpha1.OperatorPipeline) (bool, error) {
	key := types.NamespacedName{
		Namespace: pipeline.Namespace,
		Name:      certifiedIndex,
	}
	log := r.Log.WithValues("certificedimagestream", key)

	stream := newImageStream(key)
	if objects.IsObjectFound(ctx, r.Client, key, stream) {
		log.Info("existing certified image stream found")
		return false, nil // Existing ImageStream found, do nothing...
	}

	imgImport := newImageStreamImport(key)
	imgImport.Spec.Import = true
	imgImport.Spec.Repository = &imagev1.RepositoryImportSpec{
		From: corev1.ObjectReference{
			Kind: "DockerImage",
			Name: "registry.redhat.io/redhat/certified-operator-index",
		},
		ImportPolicy: imagev1.TagImportPolicy{
			Scheduled: true,
		},
		ReferencePolicy: imagev1.TagReferencePolicy{
			Type: imagev1.LocalTagReferencePolicy,
		},
	}

	log.Info("creating new certified image stream import")
	if err := r.Client.Create(ctx, imgImport); err != nil {
		return true, err
	}

	return false, nil
}

// newImageStream will create and return a new ImageStream instance using the given Name/Namespace.
func newImageStream(key types.NamespacedName) *imagev1.ImageStream {
	return &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}
}

// newImageStreamImport will create and return a new ImageStreamImport instance using the given Name/Namespace.
func newImageStreamImport(key types.NamespacedName) *imagev1.ImageStreamImport {
	return &imagev1.ImageStreamImport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}
}
