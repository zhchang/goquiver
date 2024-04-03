package k8s

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	aeclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// var dynClient *dynamic.DynamicClient
var clientset *kubernetes.Clientset
var aeClientset *aeclientset.Clientset
var once sync.Once

func Init() error {
	var err error
	once.Do(func() {
		var config *rest.Config
		if config, err = rest.InClusterConfig(); err != nil {
			// If in-cluster config fails, fall back to default kubeconfig path
			kubeconfigPath := os.Getenv("KUBECONFIG")
			if kubeconfigPath == "" {
				kubeconfigPath = clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()
			}
			// Build config from a kubeconfig filepath
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
			if err != nil {
				return
			}
		}
		config.QPS = 100
		config.Burst = 500
		// if dynClient, err = dynamic.NewForConfig(config); err != nil {
		// 	return
		// }
		if clientset, err = kubernetes.NewForConfig(config); err != nil {
			return
		}
		if aeClientset, err = aeclientset.NewForConfig(config); err != nil {
			return
		}
	})
	return err
}

func decodeYAMLToObject(yamlContent string) (*unstructured.Unstructured, error) {
	decUnstructured := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	var err error
	obj := &unstructured.Unstructured{}
	if _, _, err = decUnstructured.Decode([]byte(yamlContent), nil, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func GenManifest(ctx context.Context, chartPath string, values map[string]any) ([]K8sResource, error) {
	var err error
	actionConfig := new(action.Configuration)
	if err = actionConfig.Init(cli.New().RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), func(format string, v ...interface{}) { fmt.Printf(format, v) }); err != nil {
		return nil, err
	}

	// Load the chart
	var chart *chart.Chart
	if chart, err = loader.Load(chartPath); err != nil {
		return nil, err
	}

	// Setup install action
	install := action.NewInstall(actionConfig)
	install.ReleaseName = "whocares"
	install.DryRun = true
	install.Replace = true    // This simulates a reinstall if necessary
	install.ClientOnly = true // Since this is just for generating templates, no need for a Kubernetes cluster

	// Generate the manifest from the chart and values
	var rel *release.Release
	if rel, err = install.Run(chart, values); err != nil {
		return nil, err
	}

	// Split and output the manifest
	manifest := strings.TrimSpace(rel.Manifest)
	docs := strings.Split(manifest, "---")
	var results []K8sResource
	for _, doc := range docs {
		// Trim spaces and skip if empty
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		var obj K8sResource
		// Decode the YAML to a Kubernetes object
		if obj, err = decodeYAMLToObject(doc); err != nil {
			return nil, err
		}
		results = append(results, obj)
	}
	return results, nil
}

func GetPvcs(ctx context.Context, stsName, namespace string) ([]v1.PersistentVolumeClaim, error) {
	var err error
	if err = Init(); err != nil {
		return nil, err
	}
	var sts *appsv1.StatefulSet
	if sts, err = clientset.AppsV1().StatefulSets(namespace).Get(ctx, stsName, metav1.GetOptions{}); err != nil {
		return nil, err
	}
	var pvcs *v1.PersistentVolumeClaimList
	if pvcs, err = clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{}); err != nil {
		return nil, err
	}
	var patterns []string
	for _, vct := range sts.Spec.VolumeClaimTemplates {
		patterns = append(patterns, fmt.Sprintf("%s-%s-", vct.ObjectMeta.Name, stsName))
	}
	var result []v1.PersistentVolumeClaim
	for _, pvc := range pvcs.Items {
		for _, pattern := range patterns {
			if strings.Contains(pvc.Name, pattern) {
				result = append(result, pvc)
			}
		}
	}
	return result, nil
}

type OperationOptions struct {
	wait time.Duration
}

func (o *OperationOptions) Clone(enableWait bool) *OperationOptions {
	r := &OperationOptions{}
	if enableWait {
		r.wait = o.wait
	}
	return r
}

type OperationOption func(*OperationOptions)

func WithWait(duration time.Duration) OperationOption {
	return func(o *OperationOptions) {
		o.wait = duration
	}
}

func waitForIt[T K8sResource](ctx context.Context, api K8sAPI[T], name string, duration time.Duration) error {
	done := make(chan error)
	timeout, cancelFunc := context.WithTimeout(ctx, duration)
	defer cancelFunc()
	go func() {
		defer close(done)
		var err error
		var intf any
		for {
			if intf, err = api.Get(context.Background(), name, metav1.GetOptions{}); err != nil {
				done <- err
			}

			switch v := intf.(type) {
			case *appsv1.Deployment:
				targetReplicas := int32(1)
				if v.Spec.Replicas != nil {
					targetReplicas = *v.Spec.Replicas
				}
				if v.Status.ObservedGeneration >= v.Generation &&
					v.Status.ReadyReplicas == targetReplicas &&
					v.Status.AvailableReplicas == targetReplicas &&
					v.Status.UpdatedReplicas == targetReplicas {
					return
				} else {
					time.Sleep(1 * time.Second)
				}
			case *appsv1.StatefulSet:
				targetReplicas := int32(1)
				if v.Spec.Replicas != nil {
					targetReplicas = *v.Spec.Replicas
				}
				if v.Status.ObservedGeneration >= v.Generation &&
					v.Status.ReadyReplicas == targetReplicas &&
					v.Status.AvailableReplicas == targetReplicas &&
					v.Status.UpdatedReplicas == targetReplicas {
					return
				} else {
					time.Sleep(1 * time.Second)
				}
			default:
				return
			}

		}
	}()
	select {
	case err := <-done:
		return err
	case <-timeout.Done():
		return timeout.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}

func ensureResource[T K8sResource](r K8sResource) (T, error) {
	var t T = newResource[T]()
	switch v := r.(type) {
	case *unstructured.Unstructured:
		var err error
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(v.UnstructuredContent(), t); err != nil {
			return t, err
		}
		return t, nil
	case T:
		return v, nil
	default:
		return t, fmt.Errorf("unsupported type casting: %s", reflect.TypeOf(r).String())
	}
}

func newResource[T K8sResource]() T {
	var t T
	return reflect.New(reflect.TypeOf(t).Elem()).Interface().(T)
}

func rollout[T K8sResource](ctx context.Context, item K8sResource, apiGetter K8sAPIGetter[T], ops OperationOptions) error {
	var err error
	var instance T
	if instance, err = ensureResource[T](item); err != nil {
		return err
	}
	api := apiGetter(instance.GetNamespace())
	if _, err = api.Create(ctx, instance, metav1.CreateOptions{}); err != nil {
		if errors.IsAlreadyExists(err) {
			if _, err = api.Update(ctx, instance, metav1.UpdateOptions{}); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	if ops.wait > 0 {
		if err = waitForIt(ctx, api, instance.GetName(), ops.wait); err != nil {
			return err
		}
	}
	return nil
}

func Delete(ctx context.Context, name, namespace, kind string) error {
	panic("implement me")
}

func Rollout(ctx context.Context, item K8sResource, options ...OperationOption) error {
	var err error
	if err = Init(); err != nil {
		return err
	}
	var ops OperationOptions
	for _, option := range options {
		option(&ops)
	}
	kind := item.GetObjectKind().GroupVersionKind().Kind
	switch kind {
	case Deployment:
		return rollout[*appsv1.Deployment](ctx, item, func(namespace string) K8sAPI[*appsv1.Deployment] {
			return clientset.AppsV1().Deployments(namespace)
		}, *ops.Clone(true))
	case StatefulSet:
		return rollout[*appsv1.StatefulSet](ctx, item, func(namespace string) K8sAPI[*appsv1.StatefulSet] {
			return clientset.AppsV1().StatefulSets(namespace)
		}, *ops.Clone(true))
	case ConfigMap:
		return rollout[*v1.ConfigMap](ctx, item, func(namespace string) K8sAPI[*v1.ConfigMap] {
			return clientset.CoreV1().ConfigMaps(namespace)
		}, *ops.Clone(false))
	case CronJob:
		return rollout[*batchv1.CronJob](ctx, item, func(namespace string) K8sAPI[*batchv1.CronJob] {
			return clientset.BatchV1().CronJobs(namespace)
		}, *ops.Clone(false))
	case Service:
		return rollout[*v1.Service](ctx, item, func(namespace string) K8sAPI[*v1.Service] {
			return clientset.CoreV1().Services(namespace)
		}, *ops.Clone(false))
	case Ingress:
		return rollout[*networkingv1.Ingress](ctx, item, func(namespace string) K8sAPI[*networkingv1.Ingress] {
			return clientset.NetworkingV1().Ingresses(namespace)
		}, *ops.Clone(false))
	case PodDisruptionBudget:
		return rollout[*policyv1.PodDisruptionBudget](ctx, item, func(namespace string) K8sAPI[*policyv1.PodDisruptionBudget] {
			return clientset.PolicyV1().PodDisruptionBudgets(namespace)
		}, *ops.Clone(false))
	case Secret:
		return rollout[*v1.Secret](ctx, item, func(namespace string) K8sAPI[*v1.Secret] {
			return clientset.CoreV1().Secrets(namespace)
		}, *ops.Clone(false))
	case StorageClass:
		return rollout[*storagev1.StorageClass](ctx, item, func(_ string) K8sAPI[*storagev1.StorageClass] {
			return clientset.StorageV1().StorageClasses()
		}, *ops.Clone(false))
	case PersistentVolumeClaim:
		return rollout[*v1.PersistentVolumeClaim](ctx, item, func(namespace string) K8sAPI[*v1.PersistentVolumeClaim] {
			return clientset.CoreV1().PersistentVolumeClaims(namespace)
		}, *ops.Clone(false))
	case PersistentVolume:
		return rollout[*v1.PersistentVolume](ctx, item, func(_ string) K8sAPI[*v1.PersistentVolume] {
			return clientset.CoreV1().PersistentVolumes()
		}, *ops.Clone(false))
	case CustomResourceDefinition:
		return rollout[*apiextensionsv1.CustomResourceDefinition](ctx, item, func(_ string) K8sAPI[*apiextensionsv1.CustomResourceDefinition] {
			return aeClientset.ApiextensionsV1().CustomResourceDefinitions()
		}, *ops.Clone(false))
	case ServiceAccount:
		return rollout[*v1.ServiceAccount](ctx, item, func(namespace string) K8sAPI[*v1.ServiceAccount] {
			return clientset.CoreV1().ServiceAccounts(namespace)
		}, *ops.Clone(false))
	case ClusterRole:
		return rollout[*rbacv1.ClusterRole](ctx, item, func(_ string) K8sAPI[*rbacv1.ClusterRole] {
			return clientset.RbacV1().ClusterRoles()
		}, *ops.Clone(false))
	case ClusterRoleBinding:
		return rollout[*rbacv1.ClusterRoleBinding](ctx, item, func(_ string) K8sAPI[*rbacv1.ClusterRoleBinding] {
			return clientset.RbacV1().ClusterRoleBindings()
		}, *ops.Clone(false))
	case Role:
		return rollout[*rbacv1.Role](ctx, item, func(namespace string) K8sAPI[*rbacv1.Role] {
			return clientset.RbacV1().Roles(namespace)
		}, *ops.Clone(false))
	case RoleBinding:
		return rollout[*rbacv1.RoleBinding](ctx, item, func(namespace string) K8sAPI[*rbacv1.RoleBinding] {
			return clientset.RbacV1().RoleBindings(namespace)
		}, *ops.Clone(false))
	case DaemonSet:
		return rollout[*appsv1.DaemonSet](ctx, item, func(namespace string) K8sAPI[*appsv1.DaemonSet] {
			return clientset.AppsV1().DaemonSets(namespace)
		}, *ops.Clone(false))
	case Pod:
		return rollout[*v1.Pod](ctx, item, func(namespace string) K8sAPI[*v1.Pod] {
			return clientset.CoreV1().Pods(namespace)
		}, *ops.Clone(false))
	case Job:
		return rollout[*batchv1.Job](ctx, item, func(namespace string) K8sAPI[*batchv1.Job] {
			return clientset.BatchV1().Jobs(namespace)
		}, *ops.Clone(false))

	default:
		return fmt.Errorf("unsupported kind: %s", kind)
	}
}
