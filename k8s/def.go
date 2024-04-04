package k8s

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Kind = string

const (
	KindDeployment               Kind = "Deployment"
	KindStatefulSet              Kind = "StatefulSet"
	KindConfigMap                Kind = "ConfigMap"
	KindCronJob                  Kind = "CronJob"
	KindService                  Kind = "Service"
	KindIngress                  Kind = "Ingress"
	KindPodDisruptionBudget      Kind = "PodDisruptionBudget"
	KindSecret                   Kind = "Secret"
	KindStorageClass             Kind = "StorageClass"
	KindPersistentVolumeClaim    Kind = "PersistentVolumeClaim"
	KindPersistentVolume         Kind = "PersistentVolume"
	KindCustomResourceDefinition Kind = "CustomResourceDefinition"
	KindServiceAccount           Kind = "ServiceAccount"
	KindClusterRole              Kind = "ClusterRole"
	KindClusterRoleBinding       Kind = "ClusterRoleBinding"
	KindRole                     Kind = "Role"
	KindRoleBinding              Kind = "RoleBinding"
	KindDaemonSet                Kind = "DaemonSet"
	KindPod                      Kind = "Pod"
	KindJob                      Kind = "Job"
)

type Deployment = appsv1.Deployment
type StatefulSet = appsv1.StatefulSet
type ConfigMap = v1.ConfigMap
type CronJob = batchv1.CronJob
type Service = v1.Service
type Ingress = networkingv1.Ingress
type PodDisruptionBudget = policyv1.PodDisruptionBudget
type Secret = v1.Secret
type StorageClass = storagev1.StorageClass
type PersistentVolumeClaim = v1.PersistentVolumeClaim
type PersistentVolume = v1.PersistentVolume
type CustomResourceDefinition = apiextensionsv1.CustomResourceDefinition
type ServiceAccount = v1.ServiceAccount
type ClusterRole = rbacv1.ClusterRole
type ClusterRoleBinding = rbacv1.ClusterRoleBinding
type Role = rbacv1.Role
type RoleBinding = rbacv1.RoleBinding
type DaemonSet = appsv1.DaemonSet
type Pod = v1.Pod
type Job = batchv1.Job

type Resource interface {
	GetNamespace() string
	GetName() string
	GetObjectKind() schema.ObjectKind
}

type API[T Resource] interface {
	Create(context.Context, T, metav1.CreateOptions) (T, error)
	Update(context.Context, T, metav1.UpdateOptions) (T, error)
	Delete(context.Context, string, metav1.DeleteOptions) error
	Get(context.Context, string, metav1.GetOptions) (T, error)
}
