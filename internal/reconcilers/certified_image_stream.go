package reconcilers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	"github.com/redhat-openshift-ecosystem/operator-certification-operator/internal/objects"
	"github.com/redhat-openshift-ecosystem/operator-certification-operator/internal/pyxis"

	"github.com/go-logr/logr"
	imagev1 "github.com/openshift/api/image/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	certifiedIndex = "certified-operator-index"
)

type CertifiedImageStreamReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	pyxisClient *pyxis.PyxisClient
}

func NewCertifiedImageStreamReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *CertifiedImageStreamReconciler {
	return &CertifiedImageStreamReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
		pyxisClient: pyxis.NewPyxisClient(
			pyxis.DefaultPyxisHost,
			&http.Client{Timeout: 60 * time.Second}),
	}
}

// reconcileCertifiedImageStream will ensure that the certified operator ImageStream is present and up to date.
func (r *CertifiedImageStreamReconciler) Reconcile(ctx context.Context, pipeline *v1alpha1.OperatorPipeline) (bool, error) {
	operatorIndices, err := r.pyxisClient.FindOperatorIndices(ctx, "certified-operators")
	if err != nil {
		return true, err
	}

	key := types.NamespacedName{
		Namespace: pipeline.Namespace,
		Name:      certifiedIndex,
	}
	log := r.Log.WithValues("certificedimagestream", key)

	stream := newImageStream(key)
	if objects.IsObjectFound(ctx, r.Client, key, stream) {
		log.Info("existing certified image stream found")

		// setting owner reference on ImageStream CR, so CR gets garbage collected on OperatorPipeline deletion.
		// ignoring error, since we do not need/want to requeue on this failure,
		// and this should self correct on subsequent reconciles.
		err := controllerutil.SetControllerReference(pipeline, stream, r.Scheme)
		if err != nil {
			log.Info("unable to set owner on certified image stream, "+
				"this resource will need to be cleaned up manually on uninstall", "error", err.Error())
			return false, nil
		}
		_ = r.Update(ctx, stream)

		return false, nil // Existing ImageStream found, do nothing...
	}

	imgImport := newImageStreamImport(key)
	imgImport.Spec.Import = true

	imageSpecs := make([]imagev1.ImageImportSpec, 0, len(operatorIndices))

	for _, index := range operatorIndices {
		imageSpec := imagev1.ImageImportSpec{
			From: corev1.ObjectReference{
				Kind: "DockerImage",
				Name: fmt.Sprintf("%s:v%s", "registry.redhat.io/redhat/certified-operator-index", index.OCPVersion),
			},
			ImportPolicy: imagev1.TagImportPolicy{
				Scheduled: true,
			},
			ReferencePolicy: imagev1.TagReferencePolicy{
				Type: imagev1.LocalTagReferencePolicy,
			},
		}
		imageSpecs = append(imageSpecs, imageSpec)
	}

	imgImport.Spec.Images = imageSpecs

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
