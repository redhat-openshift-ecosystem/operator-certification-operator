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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	marketplaceIndex = "redhat-marketplace-index"
)

type MarketplaceImageStreamReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	pyxisClient *pyxis.PyxisClient
}

func NewMarketplaceImageStreamReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *MarketplaceImageStreamReconciler {
	return &MarketplaceImageStreamReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
		pyxisClient: pyxis.NewPyxisClient(
			pyxis.DefaultPyxisHost,
			&http.Client{Timeout: 60 * time.Second}),
	}
}

// reconcileMarketplaceImageStream will ensure that the Red Hat Marketplace ImageStream is present and up to date.
func (r *MarketplaceImageStreamReconciler) Reconcile(ctx context.Context, pipeline *v1alpha1.OperatorPipeline) (bool, error) {
	operatorIndices, err := r.pyxisClient.FindOperatorIndices(ctx, "redhat-marketplace")
	if err != nil {
		return true, err
	}

	key := types.NamespacedName{
		Namespace: pipeline.Namespace,
		Name:      marketplaceIndex,
	}
	log := r.Log.WithValues("marketplaceimagestream", key)

	stream := newImageStream(key)
	if objects.IsObjectFound(ctx, r.Client, key, stream) {
		log.Info("existing marketplace image stream found")

		// setting owner reference on ImageStream CR, so CR gets garbage collected on OperatorPipeline deletion.
		// ignoring error, since we do not need/want to requeue on this failure,
		// and this should self correct on subsequent reconciles.
		err := controllerutil.SetControllerReference(pipeline, stream, r.Scheme)
		if err != nil {
			log.Info("unable to set owner on marketplace image stream, "+
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
				Name: fmt.Sprintf("%s:v%s", "registry.redhat.io/redhat/redhat-marketplace-index", index.OCPVersion),
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

	log.Info("creating new marketplace image stream import")
	if err := r.Create(ctx, imgImport); err != nil {
		return true, err
	}

	return false, nil
}
