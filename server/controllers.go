package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	filteredcache "github.com/IBM/controller-filtered-cache/filteredcache"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	policyv1 "open-cluster-management.io/governance-policy-propagator/api/v1"
	placementrulev1 "open-cluster-management.io/multicloud-operators-subscription/pkg/apis/apps/placementrule/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/clyang82/multicluster-global-hub-lite/server/controllers"
)

const (
// rootPolicyLabel    = "policy.open-cluster-management.io/root-policy"
// localResourceLabel = "multicluster-global-hub.open-cluster-management.io/local-resource"
)

//go:embed manifests
var crdManifestsFS embed.FS

func (s *GlobalHubApiServer) CreateCache(ctx context.Context, cfg *rest.Config) error {

	scheme := runtime.NewScheme()

	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		return err
	}
	// if err := policyv1.AddToScheme(scheme); err != nil {
	// 	return err
	// }
	// if err := placementrulev1.AddToScheme(scheme); err != nil {
	// 	return err
	// }

	gvkLabelsMap := map[schema.GroupVersionKind][]filteredcache.Selector{
		apiextensionsv1.SchemeGroupVersion.WithKind("CustomResourceDefinition"): {
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "policies.policy.open-cluster-management.io")},
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "placementbindings.policy.open-cluster-management.io")},
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "placementrules.apps.open-cluster-management.io")},
			{FieldSelector: fmt.Sprintf("metadata.name==%s", "managedclusters.cluster.open-cluster-management.io")},
			// {FieldSelector: fmt.Sprintf("metadata.name==%s", "subscriptionreports.apps.open-cluster-management.io")},
			// {FieldSelector: fmt.Sprintf("metadata.name==%s", "subscriptions.apps.open-cluster-management.io")},
			// {FieldSelector: fmt.Sprintf("metadata.name==%s", "subscriptionstatuses.apps.open-cluster-management.io")},
			// {FieldSelector: fmt.Sprintf("metadata.name==%s", "clusterdeployments.hive.openshift.io")},
			// {FieldSelector: fmt.Sprintf("metadata.name==%s", "machinepools.hive.openshift.io")},
			// {FieldSelector: fmt.Sprintf("metadata.name==%s", "klusterletaddonconfigs.agent.open-cluster-management.io")},
		},
		// policyv1.SchemeGroupVersion.WithKind("Policy"): {
		// 	{LabelSelector: fmt.Sprint("!" + localResourceLabel)},
		// },
		// policyv1.SchemeGroupVersion.WithKind("PlacementBinding"): {
		// 	{LabelSelector: fmt.Sprint("!" + localResourceLabel)},
		// },
		// placementrulev1.SchemeGroupVersion.WithKind("PlacementRule"): {
		// 	{LabelSelector: fmt.Sprint("!" + localResourceLabel)},
		// },
	}

	opts := cache.Options{
		Scheme: scheme,
	}

	var err error
	s.Cache, err = filteredcache.NewEnhancedFilteredCacheBuilder(gvkLabelsMap)(s.hostedConfig, opts)
	if err != nil {
		return err
	}

	go s.Cache.Start(ctx)
	s.Cache.WaitForCacheSync(ctx)
	return nil
}

func (s *GlobalHubApiServer) InstallCRDController(ctx context.Context, dynamicClient dynamic.Interface) error {

	controllerName := "global-hub-crd-controller"

	// informer, err := s.Cache.GetInformerForKind(ctx, apiextensionsv1.SchemeGroupVersion.WithKind("CustomResourceDefinition"))
	// if err != nil {
	// 	return err
	// }
	// // configure the dynamic informer event handlers
	// c := controllers.NewGenericController(ctx, controllerName, dynamicClient,
	// 	apiextensionsv1.SchemeGroupVersion.WithResource("customresourcedefinitions"), informer, s.Cache,
	// 	func() client.Object { return &apiextensionsv1.CustomResourceDefinition{} })

	s.AddPostStartHook(fmt.Sprintf("start-%s", controllerName), func(hookContext genericapiserver.PostStartHookContext) error {
		installCRDs(dynamicClient)
		// go c.Run(ctx, 1)
		return nil
	})
	return nil
}

func (s *GlobalHubApiServer) InstallPolicyController(ctx context.Context, dynamicClient dynamic.Interface) error {
	controllerName := "global-hub-policy-controller"
	informer, err := s.Cache.GetInformerForKind(ctx, policyv1.SchemeGroupVersion.WithKind("Policy"))
	if err != nil {
		return err
	}
	c := controllers.NewGenericController(ctx, controllerName, dynamicClient,
		policyv1.SchemeGroupVersion.WithResource("policies"), informer, s.Cache,
		func() client.Object { return &policyv1.Policy{} })

	s.AddPostStartHook(fmt.Sprintf("start-%s", controllerName), func(hookContext genericapiserver.PostStartHookContext) error {
		go c.Run(ctx, 1)
		return nil
	})
	return nil
}

func (s *GlobalHubApiServer) InstallPlacementRuleController(ctx context.Context, dynamicClient dynamic.Interface) error {
	controllerName := "global-hub-placementrule-controller"

	informer, err := s.Cache.GetInformerForKind(ctx, placementrulev1.SchemeGroupVersion.WithKind("PlacementRule"))
	if err != nil {
		return err
	}
	c := controllers.NewGenericController(ctx, controllerName, dynamicClient,
		placementrulev1.SchemeGroupVersion.WithResource("placementrules"), informer, s.Cache,
		func() client.Object { return &placementrulev1.PlacementRule{} })

	s.AddPostStartHook(fmt.Sprintf("start-%s", controllerName), func(hookContext genericapiserver.PostStartHookContext) error {
		go c.Run(ctx, 1)
		return nil
	})
	return nil
}

func (s *GlobalHubApiServer) InstallPlacementBindingController(ctx context.Context, dynamicClient dynamic.Interface) error {
	controllerName := "global-hub-placementbinding-controller"

	informer, err := s.Cache.GetInformerForKind(ctx, policyv1.SchemeGroupVersion.WithKind("PlacementBinding"))
	if err != nil {
		return err
	}
	c := controllers.NewGenericController(ctx, controllerName, dynamicClient,
		policyv1.SchemeGroupVersion.WithResource("placementbindings"), informer, s.Cache,
		func() client.Object { return &policyv1.PlacementBinding{} })

	s.AddPostStartHook(fmt.Sprintf("start-%s", controllerName), func(hookContext genericapiserver.PostStartHookContext) error {
		go c.Run(ctx, 1)
		return nil
	})
	return nil
}

func installCRDs(dynamicClient dynamic.Interface) error {
	return fs.WalkDir(crdManifestsFS, "manifests", func(file string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			b, err := crdManifestsFS.ReadFile(file)
			if err != nil {
				return err
			}
			obj := &unstructured.Unstructured{}
			err = yaml.Unmarshal(b, &obj)
			if err != nil {
				return err
			}
			_, err = dynamicClient.
				Resource(apiextensionsv1.SchemeGroupVersion.WithResource("customresourcedefinitions")).
				Create(context.TODO(), obj, metav1.CreateOptions{})
			if err != nil {
				// we do not support to delete or update the crds
				// if k8serrors.IsAlreadyExists(err) {
				// 	_, err = dynamicClient.
				// 		Resource(apiextensionsv1.SchemeGroupVersion.WithResource("customresourcedefinitions")).
				// 		Update(context.TODO(), obj, metav1.UpdateOptions{})
				// 	if err != nil {
				// 		return err
				// 	}
				// }
				return err
			}
		}
		return nil
	})
}
