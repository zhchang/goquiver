package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Deployment               = "Deployment"
	StatefulSet              = "StatefulSet"
	ConfigMap                = "ConfigMap"
	CronJob                  = "CronJob"
	Service                  = "Service"
	Ingress                  = "Ingress"
	PodDisruptionBudget      = "PodDisruptionBudget"
	Secret                   = "Secret"
	StorageClass             = "StorageClass"
	PersistentVolumeClaim    = "PersistentVolumeClaim"
	PersistentVolume         = "PersistentVolume"
	CustomResourceDefinition = "CustomResourceDefinition"
	ServiceAccount           = "ServiceAccount"
	ClusterRole              = "ClusterRole"
	ClusterRoleBinding       = "ClusterRoleBinding"
	Role                     = "Role"
	RoleBinding              = "RoleBinding"
	DaemonSet                = "DaemonSet"
	Pod                      = "Pod"
	Job                      = "Job"
)

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

type APIGetter[T Resource] func(namespace string) API[T]
