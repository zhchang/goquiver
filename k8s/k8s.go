package k8s

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
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

func GenManifest(ctx context.Context, chartPath string, values map[string]any) ([]Resource, error) {
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
	var results []Resource
	for _, doc := range docs {
		// Trim spaces and skip if empty
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		var obj Resource
		// Decode the YAML to a Kubernetes object
		if obj, err = decodeYAMLToObject(doc); err != nil {
			return nil, err
		}
		results = append(results, obj)
	}
	return results, nil
}

func getPvcs(ctx context.Context, stsName, namespace string) ([]*v1.PersistentVolumeClaim, error) {
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
	var result []*v1.PersistentVolumeClaim
	for _, pvc := range pvcs.Items {
		for _, pattern := range patterns {
			if strings.Contains(pvc.Name, pattern) {
				result = append(result, &pvc)
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

func waitForRemove[T Resource](ctx context.Context, api API[T], name string, duration time.Duration) error {
	done := make(chan error)
	timeout, cancelFunc := context.WithTimeout(ctx, duration)
	defer cancelFunc()
	go func() {
		defer close(done)
		var err error
		var ok bool
		for {
			_, err = api.Get(context.Background(), name, metav1.GetOptions{})
			if err != nil {
				ok = errors.IsNotFound(err)
			}
			if !ok {
				time.Sleep(1 * time.Second)
				continue
			}
			return
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

func waitForRollout[T Resource](ctx context.Context, api API[T], name string, duration time.Duration) error {
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

func ensureResource[T Resource](r Resource) (T, error) {
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

func newResource[T Resource]() T {
	var t T
	return reflect.New(reflect.TypeOf(t).Elem()).Interface().(T)
}

func rollout[T Resource](ctx context.Context, item Resource, api API[T], ops OperationOptions) error {
	var err error
	var instance T
	if instance, err = ensureResource[T](item); err != nil {
		return err
	}
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
		if err = waitForRollout(ctx, api, instance.GetName(), ops.wait); err != nil {
			return err
		}
	}
	return nil
}

func remove[T Resource](ctx context.Context, name string, api API[T], ops OperationOptions) error {
	var err error
	if err = api.Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	if ops.wait > 0 {
		if err = waitForRemove(ctx, api, name, ops.wait); err != nil {
			return err
		}
	}
	return nil
}

func Remove(ctx context.Context, name, namespace, kind string, options ...OperationOption) error {
	var err error
	if err = Init(); err != nil {
		return err
	}
	var ops OperationOptions
	for _, option := range options {
		option(&ops)
	}
	switch kind {
	case KindDeployment:
		return remove[*appsv1.Deployment](ctx, name, clientset.AppsV1().Deployments(namespace), ops)
	case KindStatefulSet:
		if err = remove[*appsv1.StatefulSet](ctx, name, clientset.AppsV1().StatefulSets(namespace), ops); err != nil {
			return err
		}
		var pvcs []*v1.PersistentVolumeClaim
		if pvcs, err = getPvcs(ctx, name, namespace); err != nil {
			return err
		}
		for _, pvc := range pvcs {
			//ignore err here so that we remove all pvcs belong to statefulset in best effort
			_ = remove[*v1.PersistentVolumeClaim](ctx, pvc.GetName(), clientset.CoreV1().PersistentVolumeClaims(namespace), ops)
		}
		return nil
	case KindConfigMap:
		return remove[*v1.ConfigMap](ctx, name, clientset.CoreV1().ConfigMaps(namespace), ops)
	case KindCronJob:
		return remove[*batchv1.CronJob](ctx, name, clientset.BatchV1().CronJobs(namespace), ops)
	case KindService:
		return remove[*v1.Service](ctx, name, clientset.CoreV1().Services(namespace), ops)
	case KindIngress:
		return remove[*networkingv1.Ingress](ctx, name, clientset.NetworkingV1().Ingresses(namespace), ops)
	case KindPodDisruptionBudget:
		return remove[*policyv1.PodDisruptionBudget](ctx, name, clientset.PolicyV1().PodDisruptionBudgets(namespace), ops)
	case KindSecret:
		return remove[*v1.Secret](ctx, name, clientset.CoreV1().Secrets(namespace), ops)
	case KindStorageClass:
		return remove[*storagev1.StorageClass](ctx, name, clientset.StorageV1().StorageClasses(), ops)
	case KindPersistentVolumeClaim:
		return remove[*v1.PersistentVolumeClaim](ctx, name, clientset.CoreV1().PersistentVolumeClaims(namespace), ops)
	case KindPersistentVolume:
		return remove[*v1.PersistentVolume](ctx, name, clientset.CoreV1().PersistentVolumes(), ops)
	case KindCustomResourceDefinition:
		return remove[*apiextensionsv1.CustomResourceDefinition](ctx, name, aeClientset.ApiextensionsV1().CustomResourceDefinitions(), ops)
	case KindServiceAccount:
		return remove[*v1.ServiceAccount](ctx, name, clientset.CoreV1().ServiceAccounts(namespace), ops)
	case KindClusterRole:
		return remove[*rbacv1.ClusterRole](ctx, name, clientset.RbacV1().ClusterRoles(), ops)
	case KindClusterRoleBinding:
		return remove[*rbacv1.ClusterRoleBinding](ctx, name, clientset.RbacV1().ClusterRoleBindings(), ops)
	case KindRole:
		return remove[*rbacv1.Role](ctx, name, clientset.RbacV1().Roles(namespace), ops)
	case KindRoleBinding:
		return remove[*rbacv1.RoleBinding](ctx, name, clientset.RbacV1().RoleBindings(namespace), ops)
	case KindDaemonSet:
		return remove[*appsv1.DaemonSet](ctx, name, clientset.AppsV1().DaemonSets(namespace), ops)
	case KindPod:
		return remove[*v1.Pod](ctx, name, clientset.CoreV1().Pods(namespace), ops)
	case KindJob:
		return remove[*batchv1.Job](ctx, name, clientset.BatchV1().Jobs(namespace), ops)
	default:
		return fmt.Errorf("[remove] unsupported kind: %s", kind)
	}
}

func Rollout(ctx context.Context, item Resource, options ...OperationOption) error {
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
	case KindDeployment:
		return rollout[*appsv1.Deployment](ctx, item, clientset.AppsV1().Deployments(item.GetNamespace()), *ops.Clone(true))
	case KindStatefulSet:
		return rollout[*appsv1.StatefulSet](ctx, item, clientset.AppsV1().StatefulSets(item.GetNamespace()), *ops.Clone(true))
	case KindConfigMap:
		return rollout[*v1.ConfigMap](ctx, item, clientset.CoreV1().ConfigMaps(item.GetNamespace()), *ops.Clone(false))
	case KindCronJob:
		return rollout[*batchv1.CronJob](ctx, item, clientset.BatchV1().CronJobs(item.GetNamespace()), *ops.Clone(false))
	case KindService:
		return rollout[*v1.Service](ctx, item, clientset.CoreV1().Services(item.GetNamespace()), *ops.Clone(false))
	case KindIngress:
		return rollout[*networkingv1.Ingress](ctx, item, clientset.NetworkingV1().Ingresses(item.GetNamespace()), *ops.Clone(false))
	case KindPodDisruptionBudget:
		return rollout[*policyv1.PodDisruptionBudget](ctx, item, clientset.PolicyV1().PodDisruptionBudgets(item.GetNamespace()), *ops.Clone(false))
	case KindSecret:
		return rollout[*v1.Secret](ctx, item, clientset.CoreV1().Secrets(item.GetNamespace()), *ops.Clone(false))
	case KindStorageClass:
		return rollout[*storagev1.StorageClass](ctx, item, clientset.StorageV1().StorageClasses(), *ops.Clone(false))
	case KindPersistentVolumeClaim:
		return rollout[*v1.PersistentVolumeClaim](ctx, item, clientset.CoreV1().PersistentVolumeClaims(item.GetNamespace()), *ops.Clone(false))
	case KindPersistentVolume:
		return rollout[*v1.PersistentVolume](ctx, item, clientset.CoreV1().PersistentVolumes(), *ops.Clone(false))
	case KindCustomResourceDefinition:
		return rollout[*apiextensionsv1.CustomResourceDefinition](ctx, item, aeClientset.ApiextensionsV1().CustomResourceDefinitions(), *ops.Clone(false))
	case KindServiceAccount:
		return rollout[*v1.ServiceAccount](ctx, item, clientset.CoreV1().ServiceAccounts(item.GetNamespace()), *ops.Clone(false))
	case KindClusterRole:
		return rollout[*rbacv1.ClusterRole](ctx, item, clientset.RbacV1().ClusterRoles(), *ops.Clone(false))
	case KindClusterRoleBinding:
		return rollout[*rbacv1.ClusterRoleBinding](ctx, item, clientset.RbacV1().ClusterRoleBindings(), *ops.Clone(false))
	case KindRole:
		return rollout[*rbacv1.Role](ctx, item, clientset.RbacV1().Roles(item.GetNamespace()), *ops.Clone(false))
	case KindRoleBinding:
		return rollout[*rbacv1.RoleBinding](ctx, item, clientset.RbacV1().RoleBindings(item.GetNamespace()), *ops.Clone(false))
	case KindDaemonSet:
		return rollout[*appsv1.DaemonSet](ctx, item, clientset.AppsV1().DaemonSets(item.GetNamespace()), *ops.Clone(false))
	case KindPod:
		return rollout[*v1.Pod](ctx, item, clientset.CoreV1().Pods(item.GetNamespace()), *ops.Clone(false))
	case KindJob:
		return rollout[*batchv1.Job](ctx, item, clientset.BatchV1().Jobs(item.GetNamespace()), *ops.Clone(false))
	default:
		return fmt.Errorf("[rollout] unsupported kind: %s", kind)
	}
}

type ListOptions struct {
	pattern *regexp.Regexp
}

type ListOption func(*ListOptions)

func WithRegex(p *regexp.Regexp) ListOption {
	return func(o *ListOptions) {
		o.pattern = p
	}
}

func filter[T Resource](items []T, err error, options ListOptions) ([]T, error) {
	if err != nil {
		return nil, err
	}
	if options.pattern == nil {
		return items, nil
	}
	result := []T{}
	for _, item := range items {
		if options.pattern.MatchString(item.GetName()) {
			result = append(result, item)
		}
	}
	return result, nil
}

func into[T Resource, F any](from []F, err error, options ListOptions) ([]T, error, ListOptions) {
	if err != nil {
		return nil, err, options
	}
	var r []T
	var a any
	var t T
	var ok bool
	for _, f := range from {
		a = &f
		if t, ok = a.(T); !ok {
			return r, fmt.Errorf("Wrong Value Type"), options
		}
		r = append(r, t)
	}
	return r, nil, options

}

func List[T Resource](ctx context.Context, namespace string, options ...ListOption) ([]T, error) {
	var err error
	if err = Init(); err != nil {
		return nil, err
	}
	var ops ListOptions
	for _, option := range options {
		option(&ops)
	}
	var t any = newResource[T]()
	switch t.(type) {
	case *Deployment:
		return filter[T](into[T, appsv1.Deployment](func() ([]appsv1.Deployment, error, ListOptions) {
			if l, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *StatefulSet:
		return filter[T](into[T, appsv1.StatefulSet](func() ([]appsv1.StatefulSet, error, ListOptions) {
			if l, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *ConfigMap:
		return filter[T](into[T, v1.ConfigMap](func() ([]v1.ConfigMap, error, ListOptions) {
			if l, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *CronJob:
		return filter[T](into[T, batchv1.CronJob](func() ([]batchv1.CronJob, error, ListOptions) {
			if l, err := clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *Service:
		return filter[T](into[T, v1.Service](func() ([]v1.Service, error, ListOptions) {
			if l, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *Ingress:
		return filter[T](into[T, networkingv1.Ingress](func() ([]networkingv1.Ingress, error, ListOptions) {
			if l, err := clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *PodDisruptionBudget:
		return filter[T](into[T, policyv1.PodDisruptionBudget](func() ([]policyv1.PodDisruptionBudget, error, ListOptions) {
			if l, err := clientset.PolicyV1().PodDisruptionBudgets(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *Secret:
		return filter[T](into[T, v1.Secret](func() ([]v1.Secret, error, ListOptions) {
			if l, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *StorageClass:
		return filter[T](into[T, storagev1.StorageClass](func() ([]storagev1.StorageClass, error, ListOptions) {
			if l, err := clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *PersistentVolumeClaim:
		return filter[T](into[T, v1.PersistentVolumeClaim](func() ([]v1.PersistentVolumeClaim, error, ListOptions) {
			if l, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *PersistentVolume:
		return filter[T](into[T, v1.PersistentVolume](func() ([]v1.PersistentVolume, error, ListOptions) {
			if l, err := clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *CustomResourceDefinition:
		return filter[T](into[T, apiextensionsv1.CustomResourceDefinition](func() ([]apiextensionsv1.CustomResourceDefinition, error, ListOptions) {
			if l, err := aeClientset.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *ServiceAccount:
		return filter[T](into[T, v1.ServiceAccount](func() ([]v1.ServiceAccount, error, ListOptions) {
			if l, err := clientset.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *ClusterRole:
		return filter[T](into[T, rbacv1.ClusterRole](func() ([]rbacv1.ClusterRole, error, ListOptions) {
			if l, err := clientset.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *ClusterRoleBinding:
		return filter[T](into[T, rbacv1.ClusterRoleBinding](func() ([]rbacv1.ClusterRoleBinding, error, ListOptions) {
			if l, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *Role:
		return filter[T](into[T, rbacv1.Role](func() ([]rbacv1.Role, error, ListOptions) {
			if l, err := clientset.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *RoleBinding:
		return filter[T](into[T, rbacv1.RoleBinding](func() ([]rbacv1.RoleBinding, error, ListOptions) {
			if l, err := clientset.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *DaemonSet:
		return filter[T](into[T, appsv1.DaemonSet](func() ([]appsv1.DaemonSet, error, ListOptions) {
			if l, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *Pod:
		return filter[T](into[T, v1.Pod](func() ([]v1.Pod, error, ListOptions) {
			if l, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	case *Job:
		return filter[T](into[T, batchv1.Job](func() ([]batchv1.Job, error, ListOptions) {
			if l, err := clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{}); err == nil {
				return l.Items, nil, ops
			} else {
				return nil, err, ops
			}
		}()))
	default:
		return nil, fmt.Errorf("[list] unsupported type %v", reflect.TypeOf(t))
	}
}

func intoGet[T Resource, F Resource](f F, err error) (T, error) {
	var t T
	if err != nil {
		return t, err
	}
	var a any = f
	var ok bool
	if t, ok = a.(T); !ok {
		return t, fmt.Errorf("Wrong Value Type")
	}
	return t, nil
}

func Get[T Resource](ctx context.Context, name, namespace string) (T, error) {
	var err error
	var r T
	if err = Init(); err != nil {
		return r, err
	}
	var t any = newResource[T]()
	switch t.(type) {
	case *Deployment:
		return intoGet[T, *Deployment](clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *StatefulSet:
		return intoGet[T, *StatefulSet](clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *ConfigMap:
		return intoGet[T, *ConfigMap](clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *CronJob:
		return intoGet[T, *CronJob](clientset.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *Service:
		return intoGet[T, *Service](clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *Ingress:
		return intoGet[T, *Ingress](clientset.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *PodDisruptionBudget:
		return intoGet[T, *PodDisruptionBudget](clientset.PolicyV1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *Secret:
		return intoGet[T, *Secret](clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *StorageClass:
		return intoGet[T, *StorageClass](clientset.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{}))
	case *PersistentVolumeClaim:
		return intoGet[T, *PersistentVolumeClaim](clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *PersistentVolume:
		return intoGet[T, *PersistentVolume](clientset.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{}))
	case *CustomResourceDefinition:
		return intoGet[T, *CustomResourceDefinition](aeClientset.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{}))
	case *ServiceAccount:
		return intoGet[T, *ServiceAccount](clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *ClusterRole:
		return intoGet[T, *ClusterRole](clientset.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{}))
	case *ClusterRoleBinding:
		return intoGet[T, *ClusterRoleBinding](clientset.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{}))
	case *Role:
		return intoGet[T, *Role](clientset.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *RoleBinding:
		return intoGet[T, *RoleBinding](clientset.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *DaemonSet:
		return intoGet[T, *DaemonSet](clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *Pod:
		return intoGet[T, *Pod](clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{}))
	case *Job:
		return intoGet[T, *Job](clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{}))
	default:
		return r, fmt.Errorf("[get] unsupported type %v", reflect.TypeOf(t))
	}
}
