package controllers

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

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

func (r *OperatorPipelineReconciler) reconcilePipelineDependencies() error {

	filename := "test.yaml"

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Log.Info("Couldn't read manifest file", "File:", filename)
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
				unstructuredObj.SetNamespace("default")
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
