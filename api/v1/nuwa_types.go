/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NuwaSpec defines the desired state of Nuwa
type NuwaSpec struct {
	// Image is the container image to deploy
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// Replicas is the number of desired replicas
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Env is a list of environment variables
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Ports defines the port mappings for the service
	// +optional
	Ports []PortMapping `json:"ports,omitempty"`

	// Storage defines persistent volume configuration
	// +optional
	Storage *StorageSpec `json:"storage,omitempty"`

	// Resources defines compute resources for the container
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// ResourcesPreset defines a preset resource configuration (ignored if Resources is set)
	// +kubebuilder:validation:Enum=none;nano;micro;small;medium;large;xlarge;xxlarge
	// +kubebuilder:default=none
	// +optional
	ResourcesPreset ResourcesPreset `json:"resourcesPreset,omitempty"`

	// ResourcesOvercommit defines the overcommit level for resources
	// - none: requests = limits (no overcommit)
	// - low: requests = limits / 2 (1x overcommit)
	// - high: requests = limits / 4 (2x overcommit)
	// +kubebuilder:validation:Enum=none;low;high
	// +kubebuilder:default=none
	// +optional
	ResourcesOvercommit ResourcesOvercommit `json:"resourcesOvercommit,omitempty"`

	// Command overrides the container entrypoint
	// +optional
	Command []string `json:"command,omitempty"`

	// Args are arguments to the entrypoint
	// +optional
	Args []string `json:"args,omitempty"`

	// ImagePullPolicy defines when to pull the image
	// +kubebuilder:default=IfNotPresent
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets are secrets for pulling private images
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Service defines the Service configuration
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// UpdateStrategy defines the update strategy for the CloneSet
	// +optional
	UpdateStrategy *UpdateStrategy `json:"updateStrategy,omitempty"`
}

// ResourcesPreset defines preset resource configurations
// +kubebuilder:validation:Enum=none;nano;micro;small;medium;large;xlarge;xxlarge
type ResourcesPreset string

const (
	// ResourcesPresetNone - no resource limits
	ResourcesPresetNone ResourcesPreset = "none"
	// ResourcesPresetNano - CPU: 100m, Memory: 128Mi
	ResourcesPresetNano ResourcesPreset = "nano"
	// ResourcesPresetMicro - CPU: 250m, Memory: 256Mi
	ResourcesPresetMicro ResourcesPreset = "micro"
	// ResourcesPresetSmall - CPU: 500m, Memory: 512Mi
	ResourcesPresetSmall ResourcesPreset = "small"
	// ResourcesPresetMedium - CPU: 1, Memory: 1Gi
	ResourcesPresetMedium ResourcesPreset = "medium"
	// ResourcesPresetLarge - CPU: 2, Memory: 2Gi
	ResourcesPresetLarge ResourcesPreset = "large"
	// ResourcesPresetXLarge - CPU: 4, Memory: 4Gi
	ResourcesPresetXLarge ResourcesPreset = "xlarge"
	// ResourcesPresetXXLarge - CPU: 8, Memory: 8Gi
	ResourcesPresetXXLarge ResourcesPreset = "xxlarge"
)

// ResourcesOvercommit defines the overcommit level
// +kubebuilder:validation:Enum=none;low;high
type ResourcesOvercommit string

const (
	// ResourcesOvercommitNone - no overcommit, requests = limits
	ResourcesOvercommitNone ResourcesOvercommit = "none"
	// ResourcesOvercommitLow - 1x overcommit, requests = limits / 2
	ResourcesOvercommitLow ResourcesOvercommit = "low"
	// ResourcesOvercommitHigh - 2x overcommit, requests = limits / 4
	ResourcesOvercommitHigh ResourcesOvercommit = "high"
)

// PortMapping defines a port mapping for the container and service
type PortMapping struct {
	// Name is the name of the port
	// +optional
	Name string `json:"name,omitempty"`

	// ContainerPort is the port on the container
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	ContainerPort int32 `json:"containerPort"`

	// ServicePort is the port on the service (defaults to ContainerPort)
	// +optional
	ServicePort int32 `json:"servicePort,omitempty"`

	// Protocol is the protocol for the port (TCP/UDP)
	// +kubebuilder:default=TCP
	// +optional
	Protocol corev1.Protocol `json:"protocol,omitempty"`

	// NodePort is the port on each node (only for NodePort/LoadBalancer)
	// +optional
	NodePort int32 `json:"nodePort,omitempty"`
}

// ServiceSpec defines the Service configuration
type ServiceSpec struct {
	// Type is the Service type: ClusterIP, NodePort, LoadBalancer
	// +kubebuilder:default=ClusterIP
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	// +optional
	Type corev1.ServiceType `json:"type,omitempty"`

	// Annotations are the annotations to add to the Service
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// LoadBalancerClass is the class of the load balancer implementation
	// +optional
	LoadBalancerClass *string `json:"loadBalancerClass,omitempty"`

	// LoadBalancerIP is the IP to assign to the LoadBalancer (if supported)
	// +optional
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`

	// ExternalTrafficPolicy specifies the external traffic policy
	// +kubebuilder:validation:Enum=Cluster;Local
	// +optional
	ExternalTrafficPolicy corev1.ServiceExternalTrafficPolicy `json:"externalTrafficPolicy,omitempty"`
}

// UpdateStrategyType defines the type of update strategy
// +kubebuilder:validation:Enum=InPlaceIfPossible;ReCreate;InPlaceOnly
type UpdateStrategyType string

const (
	// InPlaceIfPossibleUpdateStrategy uses in-place update if possible, otherwise recreate
	InPlaceIfPossibleUpdateStrategy UpdateStrategyType = "InPlaceIfPossible"
	// ReCreateUpdateStrategy always recreates pods
	ReCreateUpdateStrategy UpdateStrategyType = "ReCreate"
	// InPlaceOnlyUpdateStrategy only uses in-place update, fails if not possible
	InPlaceOnlyUpdateStrategy UpdateStrategyType = "InPlaceOnly"
)

// UpdateStrategy defines the update strategy for CloneSet
type UpdateStrategy struct {
	// Type is the update strategy type: InPlaceIfPossible (default), ReCreate, InPlaceOnly
	// +kubebuilder:default=InPlaceIfPossible
	// +optional
	Type UpdateStrategyType `json:"type,omitempty"`

	// Partition is the number of pods that should be kept at the old version
	// +kubebuilder:default=0
	// +optional
	Partition *int32 `json:"partition,omitempty"`

	// MaxUnavailable is the maximum number of pods that can be unavailable during update
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`

	// MaxSurge is the maximum number of pods that can be scheduled above the desired number
	// +optional
	MaxSurge *intstr.IntOrString `json:"maxSurge,omitempty"`

	// GracePeriodSeconds is the grace period for in-place update
	// +kubebuilder:default=30
	// +optional
	GracePeriodSeconds *int32 `json:"gracePeriodSeconds,omitempty"`
}

// StorageType defines the type of storage
// +kubebuilder:validation:Enum=PVC;EmptyDir;HostPath
type StorageType string

const (
	StorageTypePVC      StorageType = "PVC"
	StorageTypeEmptyDir StorageType = "EmptyDir"
	StorageTypeHostPath StorageType = "HostPath"
)

// StorageSpec defines storage configuration
type StorageSpec struct {
	// Type is the storage type: PVC, EmptyDir, HostPath
	// +kubebuilder:default=PVC
	// +optional
	Type StorageType `json:"type,omitempty"`

	// MountPath is where to mount the volume in the container
	// +kubebuilder:validation:Required
	MountPath string `json:"mountPath"`

	// Size is the storage size (e.g., "10Gi"), defaults to "8Gi" for PVC in controller, optional for EmptyDir
	// +optional
	Size string `json:"size,omitempty"`

	// StorageClassName is the storage class to use (only for PVC)
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// AccessModes defines the access modes for the PVC (only for PVC)
	// +kubebuilder:default={ReadWriteOnce}
	// +optional
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// HostPath is the path on the host (only for HostPath type)
	// +optional
	HostPath string `json:"hostPath,omitempty"`

	// HostPathType is the type of HostPath volume
	// +optional
	HostPathType *corev1.HostPathType `json:"hostPathType,omitempty"`
}

// NuwaStatus defines the observed state of Nuwa
type NuwaStatus struct {
	// ObservedGeneration is the most recent generation observed
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Replicas is the current number of replicas
	Replicas int32 `json:"replicas,omitempty"`

	// ReadyReplicas is the number of ready replicas
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// AvailableReplicas is the number of available replicas
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// UpdatedReplicas is the number of pods with the updated template
	UpdatedReplicas int32 `json:"updatedReplicas,omitempty"`

	// Phase represents the current phase of the Nuwa resource
	// +optional
	Phase NuwaPhase `json:"phase,omitempty"`

	// ServiceIP is the ClusterIP of the service
	// +optional
	ServiceIP string `json:"serviceIP,omitempty"`

	// LoadBalancerIP is the external IP for LoadBalancer type service
	// +optional
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`

	// PVCStatus indicates the status of the PVC if storage type is PVC
	// +optional
	PVCStatus string `json:"pvcStatus,omitempty"`

	// Conditions represent the current state of the Nuwa resource
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// NuwaPhase represents the phase of a Nuwa resource
// +kubebuilder:validation:Enum=Pending;Running;Failed
type NuwaPhase string

const (
	NuwaPhasePending NuwaPhase = "Pending"
	NuwaPhaseRunning NuwaPhase = "Running"
	NuwaPhaseFailed  NuwaPhase = "Failed"
)

// Condition types for Nuwa
const (
	// ConditionCloneSetReady indicates whether the CloneSet is ready
	ConditionCloneSetReady = "CloneSetReady"
	// ConditionServiceReady indicates whether the Service is ready
	ConditionServiceReady = "ServiceReady"
	// ConditionPVCReady indicates whether the PVC is ready (only when storage type is PVC)
	ConditionPVCReady = "PVCReady"
	// ConditionProgressing indicates whether the resource is progressing
	ConditionProgressing = "Progressing"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Image",type=string,JSONPath=`.spec.image`
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Ready",type=integer,JSONPath=`.status.readyReplicas`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Nuwa is the Schema for the nuwas API
type Nuwa struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NuwaSpec   `json:"spec,omitempty"`
	Status NuwaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NuwaList contains a list of Nuwa
type NuwaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Nuwa `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Nuwa{}, &NuwaList{})
}
