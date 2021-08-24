package main

import (
	"context"
	"encoding/json"
	"flag"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery/cached/disk"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/kubectl/pkg/util"
)

type KubeClient struct {
	c      dynamic.Interface
	config *rest.Config
	mapper *restmapper.DeferredDiscoveryRESTMapper
}

type Metadata struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	ApiVersion string `json:"apiVersion"`
	Resource   string `json:"resource"`
	Kind       string `json:"kind"`
}

func NewKubeClient() (KubeClient, error) {
	var kubeconfig *string
	home := homedir.HomeDir()
	if home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.Parse()

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return KubeClient{}, err
	}

	// create the dynamic client
	config.Timeout = 180 * time.Second
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return KubeClient{}, err
	}
	// mapper
	cacheDir := filepath.Join(home, ".kube", "cache")
	httpCacheDir := filepath.Join(cacheDir, "http")
	discoveryCacheDir := computeDiscoverCacheDir(filepath.Join(cacheDir, "discovery"), config.Host)

	// DiscoveryClient queries API server about the resources
	cdc, err := disk.NewCachedDiscoveryClientForConfig(config, discoveryCacheDir, httpCacheDir, 10*time.Minute)
	if err != nil {
		return KubeClient{}, err
	}

	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cdc)

	KubeClient := KubeClient{
		c:      dynamicClient,
		config: config,
		mapper: mapper,
	}

	return KubeClient, err
}

func (k *KubeClient) ApplyWithNamespaceOverride(u *unstructured.Unstructured, namespaceOverride string) (Metadata, error) {
	metadata := Metadata{}
	gvk := u.GroupVersionKind()

	restMapping, err := k.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return metadata, err
	}

	gvr := restMapping.Resource
	gv := gvk.GroupVersion()
	k.config.GroupVersion = &gv

	restClient, err := NewRestClient(*k.config, gv)
	if err != nil {
		return metadata, err
	}

	helper := resource.NewHelper(restClient, restMapping)
	if namespaceOverride == "" {
		namespace := u.GetNamespace()
		if helper.NamespaceScoped && namespace == "" {
			namespace = "default"
			u.SetNamespace(namespace)
		}
	} else {
		if helper.NamespaceScoped {
			u.SetNamespace(namespaceOverride)
		}
	}

	info := &resource.Info{
		Client:          restClient,
		Mapping:         restMapping,
		Namespace:       u.GetNamespace(),
		Name:            u.GetName(),
		Source:          "",
		Object:          u,
		ResourceVersion: restMapping.Resource.Version,
	}

	patcher, err := NewPatcher(info, helper)
	if err != nil {
		return metadata, err
	}

	// Get the modified configuration of the object. Embed the result
	// as an annotation in the modified configuration, so that it will appear
	// in the patch sent to the server.
	modified, err := util.GetModifiedConfiguration(info.Object, true, unstructured.UnstructuredJSONScheme)
	if err != nil {
		return metadata, err
	}

	if err := info.Get(); err != nil {
		if !errors.IsNotFound(err) {
			return metadata, err
		}

		// Create the resource if it doesn't exist
		// First, update the annotation used by kubectl apply
		if err := util.CreateApplyAnnotation(info.Object, unstructured.UnstructuredJSONScheme); err != nil {
			return metadata, err
		}

		// Then create the resource and skip the three-way merge
		obj, err := helper.Create(info.Namespace, true, info.Object)
		if err != nil {
			return metadata, err
		}
		info.Refresh(obj, true)
	}

	_, patchedObject, err := patcher.Patch(info.Object, modified, info.Namespace, info.Name)
	if err != nil {
		return metadata, err
	}

	info.Refresh(patchedObject, true)

	metadata.Name = u.GetName()
	metadata.Namespace = u.GetNamespace()
	metadata.ApiVersion = gvr.Group + "/" + gvr.Version
	metadata.Resource = gvr.Resource
	metadata.Kind = gvk.Kind

	return metadata, nil
}

func (k *KubeClient) DeleteResourceByKindAndNameAndNamespace(kind, name, namespace string, do metav1.DeleteOptions) error {
	gvk, err := k.mapper.KindFor(schema.GroupVersionResource{Resource: kind})
	if err != nil {
		return err
	}

	restMapping, err := k.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	restClient, err := NewRestClient(*k.config, gvk.GroupVersion())
	if err != nil {
		return err
	}

	helper := resource.NewHelper(restClient, restMapping)
	if helper.NamespaceScoped {
		err = k.c.
			Resource(restMapping.Resource).
			Namespace(namespace).
			Delete(context.TODO(), name, do)
	} else {
		err = k.c.
			Resource(restMapping.Resource).
			Delete(context.TODO(), name, do)
	}

	return err
}

func NewRestClient(restConfig rest.Config, gv schema.GroupVersion) (rest.Interface, error) {
	restConfig.ContentConfig = resource.UnstructuredPlusDefaultContentConfig()
	restConfig.GroupVersion = &gv
	if len(gv.Group) == 0 {
		restConfig.APIPath = "/api"
	} else {
		restConfig.APIPath = "/apis"
	}

	return rest.RESTClientFor(&restConfig)
}

var overlyCautiousIllegalFileCharacters = regexp.MustCompile(`[^(\w/\.)]`)

// computeDiscoverCacheDir takes the parentDir and the host and comes up with a "usually non-colliding" name.
func computeDiscoverCacheDir(parentDir, host string) string {
	// strip the optional scheme from host if its there:
	schemelessHost := strings.Replace(strings.Replace(host, "https://", "", 1), "http://", "", 1)
	// now do a simple collapse of non-AZ09 characters.  Collisions are possible but unlikely.
	// Even if we do collide the problem is short lived
	safeHost := overlyCautiousIllegalFileCharacters.ReplaceAllString(schemelessHost, "_")
	return filepath.Join(parentDir, safeHost)
}

func ToUnstructured(manifest map[string]interface{}) (*unstructured.Unstructured, error) {
	b, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	return JSONToUnstructured(b)
}

func JSONToUnstructured(jsonString []byte) (*unstructured.Unstructured, error) {
	obj, _, err := unstructured.UnstructuredJSONScheme.Decode(jsonString, nil, nil)
	if err != nil {
		return nil, err
	}
	// Convert the runtime.Object to unstructured.Unstructured.
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	return &unstructured.Unstructured{
		Object: m,
	}, nil
}
