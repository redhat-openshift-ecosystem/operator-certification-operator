package controllers

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	"os"

	"path/filepath"

	git "github.com/go-git/go-git/v5"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	OPERATOR_PIPELINES_REPO = "https://github.com/redhat-openshift-ecosystem/operator-pipelines.git"
	REPO_CLONE_PATH         = "/tmp/operator-pipelines/"
	PIPELINE_MANIFESTS_PATH = "ansible/roles/operator-pipeline/templates/openshift/pipelines"
	TASKS_MANIFESTS_PATH    = "ansible/roles/operator-pipeline/templates/openshift/tasks"
)

func (r *OperatorPipelineReconciler) reconcilePipelineDependencies(meta metav1.ObjectMeta) error {

	// Cloning operator-pipelines project to retrieve pipelines and tasks
	// yaml manifests that need to be applied beforehand
	// ref: https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/ci-pipeline.md#step-6---install-the-certification-pipeline-and-dependencies-into-the-cluster

	_, err := git.PlainClone(REPO_CLONE_PATH, false, &git.CloneOptions{
		URL:      OPERATOR_PIPELINES_REPO,
		Progress: os.Stdout,
	})
	if err != nil {
		log.Log.Info("Couldn't clone the repository for operator-pipelines.")
		return err
	}
	defer r.RemovePipelineDependencyFiles(REPO_CLONE_PATH)

	// Reading pipeline manifests and applying to cluster
	root := REPO_CLONE_PATH + PIPELINE_MANIFESTS_PATH
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			if errors := r.applyManifests(path, meta.Namespace); errors != nil {
				return errors
			}
		}
		return nil
	})
	if err != nil {
		log.Log.Info("Couldn't iterate over operator-pipelines yaml manifest files")
		return err
	}

	// Reading tasks manifests and applying it to cluster
	root = REPO_CLONE_PATH + TASKS_MANIFESTS_PATH
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			if errors := r.applyManifests(path, meta.Namespace); errors != nil {
				return errors
			}
		}
		return nil
	})
	if err != nil {
		log.Log.Info("Couldn't iterate over operator-pipelines yaml manifest files")
		return err
	}

	return nil
}

func (r *OperatorPipelineReconciler) RemovePipelineDependencyFiles(filePath string) error {
	if err := os.RemoveAll(filePath); err != nil {
		log.Log.Info("Couldn't remove operator-pipelines directory")
		return err
	}
	return nil
}

func (r *OperatorPipelineReconciler) applyManifests(fileName string, Namespace string) error {

	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Log.Info("Couldn't read manifest file", "File:", fileName)
		return err
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Log.Info("Couldn't get in cluster config.")
		return err
	}

	c, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Log.Info("Couldn't initialize kubernetes client from config.")
		return err
	}

	dd, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Log.Info("Couldn't initialize dynamic k8s client from config.")
		return err
	}

	// Decoding yaml files to resource objects and apply to cluster
	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(b), 100)
	for {
		var rawObj runtime.RawExtension
		if err = decoder.Decode(&rawObj); err != nil {
			break
		}

		obj, gvk, err := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			log.Log.Info("Couldn't decode obj and gvk.")
			return err
		}
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			log.Log.Info("Coundn't convert obj to unstructured Map")
			return err
		}

		unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

		gr, err := restmapper.GetAPIGroupResources(c.Discovery())
		if err != nil {
			log.Log.Info("Couldn't get API group resources")
			return err
		}

		mapper := restmapper.NewDiscoveryRESTMapper(gr)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			log.Log.Info("Couldn't the preferred resource mapping for given kind.")
			return err
		}

		var dri dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace(Namespace)
			}
			dri = dd.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			dri = dd.Resource(mapping.Resource)
		}

		if _, err := dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{}); err != nil {
			log.Log.Info("Couldn't create resource.")
			return err
		}
	}
	if err != io.EOF {
		log.Log.Info("Error ocurred reading file.")
		return err
	}
	return nil
}
