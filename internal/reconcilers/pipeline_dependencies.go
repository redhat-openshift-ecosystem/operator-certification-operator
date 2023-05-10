package reconcilers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	securityv1 "github.com/openshift/api/security/v1"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const (
	operatorCIPipelineYml      = "operator-ci-pipeline.yml"
	operatorHostedPipelineYml  = "operator-hosted-pipeline.yml"
	operatorReleasePipelineYml = "operator-release-pipeline.yml"
	clusterRoleYml             = "openshift-pipeline-sa-scc-role.yml"
	clusterRoleBindingYml      = "openshift-pipeline-sa-scc-role-bindings.yml"
	sccYml                     = "openshift-pipelines-custom-scc.yml"
	ClusterResourceLabel       = "operatorpipelines.certification.redhat.com/cluster-resource"
	NamespaceLabel             = "operatorpipelines.certification.redhat.com/metadata.name"
)

var (
	baseManifestsPath     = filepath.Join("ansible", "roles", "operator-pipeline", "templates", "openshift")
	pipelineManifestsPath = filepath.Join(baseManifestsPath, "pipelines")
	taskManifestsPath     = filepath.Join(baseManifestsPath, "tasks")
)

type PipelineDependenciesReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func NewPipeDependenciesReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *PipelineDependenciesReconciler {
	return &PipelineDependenciesReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

func (r *PipelineDependenciesReconciler) Reconcile(ctx context.Context, pipeline *v1alpha1.OperatorPipeline) (bool, error) {
	// Cloning operator-pipelines project to retrieve pipelines and tasks
	// yaml manifests that need to be applied beforehand
	// ref: https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/ci-pipeline.md#step-6---install-the-certification-pipeline-and-dependencies-into-the-cluster
	log := r.Log.WithName("pipelinedependencies")

	gitPath := filepath.Join(os.Getenv("GIT_REPO_PATH"), "operator-pipeline")
	// This will check that the repo has been cloned and is valid
	_, err := git.PlainOpen(gitPath)
	if err != nil {
		log.Error(err, "Pipelines repo is not valid")
		return true, err
	}

	pipelineManifestsPath := filepath.Join(gitPath, pipelineManifestsPath)

	if err := r.applyOrDeletePipeline(ctx, pipeline, pipeline.Spec.ApplyCIPipeline, filepath.Join(pipelineManifestsPath, operatorCIPipelineYml)); err != nil {
		return true, err
	}

	if err := r.applyOrDeletePipeline(ctx, pipeline, pipeline.Spec.ApplyHostedPipeline, filepath.Join(pipelineManifestsPath, operatorHostedPipelineYml)); err != nil {
		return true, err
	}

	if err := r.applyOrDeletePipeline(ctx, pipeline, pipeline.Spec.ApplyReleasePipeline, filepath.Join(pipelineManifestsPath, operatorReleasePipelineYml)); err != nil {
		return true, err
	}

	taskManifestsPath := filepath.Join(gitPath, taskManifestsPath)

	tasks, err := os.ReadDir(taskManifestsPath)
	if err != nil {
		log.Error(err, "could not read tasks directory")
		return true, err
	}

	for _, task := range tasks {
		if !task.IsDir() {
			if err := r.applyManifests(ctx, filepath.Join(taskManifestsPath, task.Name()), pipeline, new(tekton.Task), false); err != nil {
				return true, err
			}
		}
	}

	if err := r.applyManifests(ctx, filepath.Join(gitPath, baseManifestsPath, sccYml), pipeline, new(securityv1.SecurityContextConstraints), true); err != nil {
		return true, err
	}

	if err := r.applyManifests(ctx, filepath.Join(gitPath, baseManifestsPath, clusterRoleYml), pipeline, new(rbacv1.ClusterRole), true); err != nil {
		return true, err
	}

	// apply the cluster role binding with modifications below
	tmpFile, err := r.modifyAndSaveTempClusterRoleBinding(ctx, filepath.Join(gitPath, baseManifestsPath, clusterRoleBindingYml), pipeline, new(rbacv1.ClusterRoleBinding))
	if err != nil {
		return true, err
	}
	defer os.Remove(tmpFile)

	if err := r.applyManifests(ctx, tmpFile, pipeline, new(rbacv1.ClusterRoleBinding), true); err != nil {
		return true, err
	}

	return false, nil
}

func (r *PipelineDependenciesReconciler) applyOrDeletePipeline(ctx context.Context, pipeline *v1alpha1.OperatorPipeline, applyManifest bool, yamlPath string) error {
	if applyManifest {
		return r.applyManifests(ctx, yamlPath, pipeline, new(tekton.Pipeline), false)
	}
	return r.deleteManifests(ctx, yamlPath, pipeline, new(tekton.Pipeline))
}

func (r *PipelineDependenciesReconciler) applyManifests(ctx context.Context, fileName string, owner, obj client.Object, addClusterResourceLabel bool) error {
	log := r.Log.WithName("applyManifests")

	b, err := os.ReadFile(fileName)
	if err != nil {
		log.Error(err, fmt.Sprintf("Couldn't read manifest file for: %s", fileName))
		return err
	}

	if err = yamlutil.Unmarshal(b, &obj); err != nil {
		log.Error(err, fmt.Sprintf("Couldn't unmarshall yaml file for: %s", fileName))
		return err
	}

	// adding cluster resource label for scc, clusterrole, clusterrolebinding, this is so they can be selected
	// by label on deletion
	if addClusterResourceLabel {
		labels := obj.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}

		labels[ClusterResourceLabel] = "true"

		obj.SetLabels(labels)
	}

	obj.SetNamespace(owner.GetNamespace())
	err = r.Client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)

	if len(obj.GetUID()) > 0 {
		if err := r.Client.Update(ctx, obj); err != nil {
			log.Error(err, fmt.Sprintf("failed to update pipeline resource for file: %s", fileName))
			return err
		}
	}

	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		_ = controllerutil.SetControllerReference(owner, obj, r.Scheme)
		if err := r.Client.Create(ctx, obj); err != nil {
			log.Error(err, fmt.Sprintf("failed to create pipeline resource for file: %s", fileName))
			return err
		}
	}

	return nil
}

func (r *PipelineDependenciesReconciler) deleteManifests(ctx context.Context, fileName string, owner, obj client.Object) error {
	log := r.Log.WithName("deleteManifests")
	b, err := os.ReadFile(fileName)
	if err != nil {
		log.Error(err, fmt.Sprintf("Couldn't read manifest file for: %s", fileName))
		return err
	}

	if err = yamlutil.Unmarshal(b, &obj); err != nil {
		log.Error(err, fmt.Sprintf("Couldn't unmarshall yaml file for: %s", fileName))
		return err
	}

	obj.SetNamespace(owner.GetNamespace())
	if err := r.Client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj); err != nil && !errors.IsNotFound(err) {
		return err
	} else if errors.IsNotFound(err) {
		return nil
	}

	if err := r.Client.Delete(ctx, obj); err != nil {
		log.Error(err, fmt.Sprintf("failed to delete pipeline resource for file: %s", fileName))
		return err
	}

	return nil
}

func (r *PipelineDependenciesReconciler) modifyAndSaveTempClusterRoleBinding(_ context.Context, fileName string, owner, obj client.Object) (string, error) {
	log := r.Log.WithName("modifyAndSaveTempClusterRoleBinding")

	b, err := os.ReadFile(fileName)
	if err != nil {
		log.Error(err, fmt.Sprintf("Couldn't read manifest file for: %s", fileName))
		return "", err
	}

	if err = yamlutil.Unmarshal(b, &obj); err != nil {
		log.Error(err, fmt.Sprintf("Couldn't unmarshal yaml file for: %s", fileName))
		return "", err
	}

	// type asserting to ensure proper kind inorder to update specific values before updating/creating in cluster
	crb, ok := obj.(*rbacv1.ClusterRoleBinding)
	if !ok {
		return "", fmt.Errorf("could not type assert client.Object as ClusterRoleBinding")
	}

	// updating the below values, since the yaml read in is templated and incomplete
	crb.Name = fmt.Sprintf("pipeline-%s-pipelines-custom-scc", owner.GetNamespace())
	crb.Subjects[0].Namespace = owner.GetNamespace()

	// adding additional label so it can be selected by label on deletion
	metav1.SetMetaDataLabel(&crb.ObjectMeta, NamespaceLabel, owner.GetNamespace())

	b, err = yaml.Marshal(crb)
	if err != nil {
		log.Error(err, fmt.Sprintf("Couldn't marshal ClusterRoleBinding for: %s", fileName))
		return "", err
	}

	// creating temp file in a temp directory
	tmpFile, err := os.CreateTemp("", "cluster-role-binding-*.yaml")
	if err != nil {
		log.Error(err, fmt.Sprintf("Couldn't create temp file for: %s", fileName))
		return "", err
	}

	_, err = tmpFile.Write(b)
	if err != nil {
		log.Error(err, fmt.Sprintf("Couldn't write manifest file for: %s", fileName))
		return "", err
	}

	return tmpFile.Name(), nil
}
