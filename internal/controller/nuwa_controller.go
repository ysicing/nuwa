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

package controller

import (
	"context"
	"fmt"

	appspub "github.com/openkruise/kruise-api/apps/pub"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appv1 "github.com/ysicing/nuwa/api/v1"
)

const (
	nuwaFinalizer = "app.12306.work/finalizer"
)

// NuwaReconciler reconciles a Nuwa object
type NuwaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=app.12306.work,resources=nuwas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.12306.work,resources=nuwas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=app.12306.work,resources=nuwas/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps.kruise.io,resources=clonesets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

func (r *NuwaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Nuwa instance
	nuwa := &appv1.Nuwa{}
	if err := r.Get(ctx, req.NamespacedName, nuwa); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !nuwa.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(nuwa, nuwaFinalizer) {
			// Perform cleanup if needed
			controllerutil.RemoveFinalizer(nuwa, nuwaFinalizer)
			if err := r.Update(ctx, nuwa); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(nuwa, nuwaFinalizer) {
		controllerutil.AddFinalizer(nuwa, nuwaFinalizer)
		if err := r.Update(ctx, nuwa); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Set initial phase
	if nuwa.Status.Phase == "" {
		nuwa.Status.Phase = appv1.NuwaPhasePending
		if err := r.Status().Update(ctx, nuwa); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Reconcile PVC first (if storage type is PVC)
	// PVC must be created before CloneSet to avoid Pod pending
	if nuwa.Spec.Storage != nil && nuwa.Spec.Storage.Type == appv1.StorageTypePVC {
		if err := r.reconcilePVC(ctx, nuwa); err != nil {
			log.Error(err, "Failed to reconcile PVC")
			return ctrl.Result{}, err
		}
	}

	// Reconcile CloneSet
	if err := r.reconcileCloneSet(ctx, nuwa); err != nil {
		log.Error(err, "Failed to reconcile CloneSet")
		return ctrl.Result{}, err
	}

	// Reconcile Service (only if ports are defined)
	if len(nuwa.Spec.Ports) > 0 {
		if err := r.reconcileService(ctx, nuwa); err != nil {
			log.Error(err, "Failed to reconcile Service")
			return ctrl.Result{}, err
		}
	}

	// Update status from CloneSet
	if err := r.updateStatus(ctx, nuwa); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NuwaReconciler) reconcileCloneSet(ctx context.Context, nuwa *appv1.Nuwa) error {
	cloneSet := r.buildCloneSet(nuwa)

	// Set owner reference
	if err := controllerutil.SetControllerReference(nuwa, cloneSet, r.Scheme); err != nil {
		return err
	}

	// Check if CloneSet exists
	found := &kruiseappsv1alpha1.CloneSet{}
	err := r.Get(ctx, types.NamespacedName{Name: cloneSet.Name, Namespace: cloneSet.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Create new CloneSet
		return r.Create(ctx, cloneSet)
	} else if err != nil {
		return err
	}

	// Update existing CloneSet
	found.Spec = cloneSet.Spec
	return r.Update(ctx, found)
}

func (r *NuwaReconciler) buildCloneSet(nuwa *appv1.Nuwa) *kruiseappsv1alpha1.CloneSet {
	replicas := int32(1)
	if nuwa.Spec.Replicas != nil {
		replicas = *nuwa.Spec.Replicas
	}

	labels := buildLabels(nuwa.Name)

	// Build container ports
	containerPorts := make([]corev1.ContainerPort, 0, len(nuwa.Spec.Ports))
	for _, p := range nuwa.Spec.Ports {
		name := p.Name
		if name == "" {
			name = fmt.Sprintf("port-%d", p.ContainerPort)
		}
		protocol := p.Protocol
		if protocol == "" {
			protocol = corev1.ProtocolTCP
		}
		containerPorts = append(containerPorts, corev1.ContainerPort{
			Name:          name,
			ContainerPort: p.ContainerPort,
			Protocol:      protocol,
		})
	}

	// Build container
	container := corev1.Container{
		Name:            "main",
		Image:           nuwa.Spec.Image,
		ImagePullPolicy: nuwa.Spec.ImagePullPolicy,
		Ports:           containerPorts,
		Env:             nuwa.Spec.Env,
		Command:         nuwa.Spec.Command,
		Args:            nuwa.Spec.Args,
	}

	// Set resources: explicit Resources takes priority over ResourcesPreset
	if nuwa.Spec.Resources != nil {
		container.Resources = *nuwa.Spec.Resources
	} else {
		container.Resources = getResourcesFromPreset(nuwa.Spec.ResourcesPreset, nuwa.Spec.ResourcesOvercommit)
	}

	// Build volume mounts and volumes based on storage type
	var volumes []corev1.Volume
	if nuwa.Spec.Storage != nil {
		container.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: nuwa.Spec.Storage.MountPath,
			},
		}

		storageType := nuwa.Spec.Storage.Type
		if storageType == "" {
			storageType = appv1.StorageTypePVC
		}

		switch storageType {
		case appv1.StorageTypePVC:
			// Use existing PVC created by reconcilePVC
			volumes = []corev1.Volume{
				{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: getPVCName(nuwa.Name),
						},
					},
				},
			}
		case appv1.StorageTypeEmptyDir:
			emptyDir := &corev1.EmptyDirVolumeSource{}
			if nuwa.Spec.Storage.Size != "" {
				emptyDir.SizeLimit = resourcePtr(resource.MustParse(nuwa.Spec.Storage.Size))
			}
			volumes = []corev1.Volume{
				{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: emptyDir,
					},
				},
			}
		case appv1.StorageTypeHostPath:
			hostPathType := corev1.HostPathDirectoryOrCreate
			if nuwa.Spec.Storage.HostPathType != nil {
				hostPathType = *nuwa.Spec.Storage.HostPathType
			}
			volumes = []corev1.Volume{
				{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: nuwa.Spec.Storage.HostPath,
							Type: &hostPathType,
						},
					},
				},
			}
		}
	}

	cloneSet := &kruiseappsv1alpha1.CloneSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nuwa.Name,
			Namespace: nuwa.Namespace,
			Labels:    labels,
		},
		Spec: kruiseappsv1alpha1.CloneSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers:       []corev1.Container{container},
					ImagePullSecrets: nuwa.Spec.ImagePullSecrets,
					Volumes:          volumes,
				},
			},
			UpdateStrategy: buildUpdateStrategy(nuwa.Spec.UpdateStrategy),
		},
	}

	return cloneSet
}

// buildUpdateStrategy builds the CloneSet update strategy
func buildUpdateStrategy(strategy *appv1.UpdateStrategy) kruiseappsv1alpha1.CloneSetUpdateStrategy {
	// Default to InPlaceIfPossible
	updateStrategy := kruiseappsv1alpha1.CloneSetUpdateStrategy{
		Type: kruiseappsv1alpha1.InPlaceIfPossibleCloneSetUpdateStrategyType,
	}

	if strategy == nil {
		return updateStrategy
	}

	// Set update type
	switch strategy.Type {
	case appv1.ReCreateUpdateStrategy:
		updateStrategy.Type = kruiseappsv1alpha1.RecreateCloneSetUpdateStrategyType
	case appv1.InPlaceOnlyUpdateStrategy:
		updateStrategy.Type = kruiseappsv1alpha1.InPlaceOnlyCloneSetUpdateStrategyType
	default:
		updateStrategy.Type = kruiseappsv1alpha1.InPlaceIfPossibleCloneSetUpdateStrategyType
	}

	// Set partition
	if strategy.Partition != nil {
		partition := intstr.FromInt32(*strategy.Partition)
		updateStrategy.Partition = &partition
	}

	// Set maxUnavailable
	if strategy.MaxUnavailable != nil {
		updateStrategy.MaxUnavailable = strategy.MaxUnavailable
	}

	// Set maxSurge
	if strategy.MaxSurge != nil {
		updateStrategy.MaxSurge = strategy.MaxSurge
	}

	// Set in-place update strategy
	if strategy.GracePeriodSeconds != nil {
		updateStrategy.InPlaceUpdateStrategy = &appspub.InPlaceUpdateStrategy{
			GracePeriodSeconds: *strategy.GracePeriodSeconds,
		}
	}

	return updateStrategy
}

func (r *NuwaReconciler) reconcileService(ctx context.Context, nuwa *appv1.Nuwa) error {
	svc := r.buildService(nuwa)

	// Set owner reference
	if err := controllerutil.SetControllerReference(nuwa, svc, r.Scheme); err != nil {
		return err
	}

	// Check if Service exists
	found := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Create new Service
		return r.Create(ctx, svc)
	} else if err != nil {
		return err
	}

	// Update existing Service (preserve ClusterIP)
	svc.Spec.ClusterIP = found.Spec.ClusterIP
	found.Spec = svc.Spec
	return r.Update(ctx, found)
}

func (r *NuwaReconciler) reconcilePVC(ctx context.Context, nuwa *appv1.Nuwa) error {
	pvc, err := r.buildPVC(nuwa)
	if err != nil {
		return err
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(nuwa, pvc, r.Scheme); err != nil {
		return err
	}

	// Check if PVC exists
	found := &corev1.PersistentVolumeClaim{}
	err = r.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Create new PVC
		return r.Create(ctx, pvc)
	} else if err != nil {
		return err
	}

	// PVC exists, no need to update (PVC spec is immutable except for size)
	// TODO: Support PVC resize if needed
	return nil
}

func (r *NuwaReconciler) buildPVC(nuwa *appv1.Nuwa) (*corev1.PersistentVolumeClaim, error) {
	labels := buildLabels(nuwa.Name)

	accessModes := nuwa.Spec.Storage.AccessModes
	if len(accessModes) == 0 {
		accessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}

	storageSize := nuwa.Spec.Storage.Size
	if storageSize == "" {
		storageSize = "8Gi"
	}

	// Validate storage size format
	quantity, err := resource.ParseQuantity(storageSize)
	if err != nil {
		return nil, fmt.Errorf("invalid storage size format %q: %w", storageSize, err)
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getPVCName(nuwa.Name),
			Namespace: nuwa.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      accessModes,
			StorageClassName: nuwa.Spec.Storage.StorageClassName,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: quantity,
				},
			},
		},
	}, nil
}

func (r *NuwaReconciler) buildService(nuwa *appv1.Nuwa) *corev1.Service {
	labels := buildLabels(nuwa.Name)

	servicePorts := make([]corev1.ServicePort, 0, len(nuwa.Spec.Ports))
	for _, p := range nuwa.Spec.Ports {
		name := p.Name
		if name == "" {
			name = fmt.Sprintf("port-%d", p.ContainerPort)
		}
		servicePort := p.ServicePort
		if servicePort == 0 {
			servicePort = p.ContainerPort
		}
		protocol := p.Protocol
		if protocol == "" {
			protocol = corev1.ProtocolTCP
		}

		sp := corev1.ServicePort{
			Name:       name,
			Port:       servicePort,
			TargetPort: intstr.FromInt32(p.ContainerPort),
			Protocol:   protocol,
		}
		if p.NodePort > 0 {
			sp.NodePort = p.NodePort
		}
		servicePorts = append(servicePorts, sp)
	}

	// Get service configuration
	serviceType := corev1.ServiceTypeClusterIP
	var annotations map[string]string
	var loadBalancerClass *string
	var loadBalancerIP string
	var externalTrafficPolicy corev1.ServiceExternalTrafficPolicy

	if nuwa.Spec.Service != nil {
		if nuwa.Spec.Service.Type != "" {
			serviceType = nuwa.Spec.Service.Type
		}
		annotations = nuwa.Spec.Service.Annotations
		loadBalancerClass = nuwa.Spec.Service.LoadBalancerClass
		loadBalancerIP = nuwa.Spec.Service.LoadBalancerIP
		externalTrafficPolicy = nuwa.Spec.Service.ExternalTrafficPolicy
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        nuwa.Name,
			Namespace:   nuwa.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Type:              serviceType,
			Selector:          labels,
			Ports:             servicePorts,
			LoadBalancerClass: loadBalancerClass,
		},
	}

	if loadBalancerIP != "" {
		svc.Spec.LoadBalancerIP = loadBalancerIP
	}
	if externalTrafficPolicy != "" {
		svc.Spec.ExternalTrafficPolicy = externalTrafficPolicy
	}

	return svc
}

func (r *NuwaReconciler) updateStatus(ctx context.Context, nuwa *appv1.Nuwa) error {
	log := logf.FromContext(ctx)

	// Update observed generation
	nuwa.Status.ObservedGeneration = nuwa.Generation

	// Get CloneSet status
	cloneSet := &kruiseappsv1alpha1.CloneSet{}
	cloneSetErr := r.Get(ctx, types.NamespacedName{Name: nuwa.Name, Namespace: nuwa.Namespace}, cloneSet)
	if cloneSetErr != nil && !errors.IsNotFound(cloneSetErr) {
		return cloneSetErr
	}

	if cloneSetErr == nil {
		// Update Nuwa status from CloneSet
		nuwa.Status.Replicas = cloneSet.Status.Replicas
		nuwa.Status.ReadyReplicas = cloneSet.Status.ReadyReplicas
		nuwa.Status.AvailableReplicas = cloneSet.Status.AvailableReplicas
		nuwa.Status.UpdatedReplicas = cloneSet.Status.UpdatedReplicas

		// Set CloneSetReady condition
		cloneSetReady := cloneSet.Status.ReadyReplicas == cloneSet.Status.Replicas && cloneSet.Status.Replicas > 0
		setCondition(nuwa, appv1.ConditionCloneSetReady, cloneSetReady,
			"CloneSetReady", "CloneSet is ready",
			"CloneSetNotReady", fmt.Sprintf("CloneSet has %d/%d ready replicas", cloneSet.Status.ReadyReplicas, cloneSet.Status.Replicas))

		// Set Progressing condition
		progressing := cloneSet.Status.UpdatedReplicas != cloneSet.Status.Replicas
		setCondition(nuwa, appv1.ConditionProgressing, progressing,
			"Progressing", fmt.Sprintf("Rolling update in progress: %d/%d updated", cloneSet.Status.UpdatedReplicas, cloneSet.Status.Replicas),
			"NotProgressing", "All replicas are up to date")
	}

	// Get Service status
	service := &corev1.Service{}
	serviceErr := r.Get(ctx, types.NamespacedName{Name: nuwa.Name, Namespace: nuwa.Namespace}, service)
	if serviceErr != nil && !errors.IsNotFound(serviceErr) {
		log.Error(serviceErr, "Failed to get Service")
	}

	if serviceErr == nil {
		nuwa.Status.ServiceIP = service.Spec.ClusterIP

		// Get LoadBalancer IP if applicable
		if service.Spec.Type == corev1.ServiceTypeLoadBalancer && len(service.Status.LoadBalancer.Ingress) > 0 {
			ingress := service.Status.LoadBalancer.Ingress[0]
			if ingress.IP != "" {
				nuwa.Status.LoadBalancerIP = ingress.IP
			} else if ingress.Hostname != "" {
				nuwa.Status.LoadBalancerIP = ingress.Hostname
			}
		}

		// Set ServiceReady condition
		setCondition(nuwa, appv1.ConditionServiceReady, true,
			"ServiceReady", "Service is ready",
			"", "")
	} else if errors.IsNotFound(serviceErr) {
		nuwa.Status.ServiceIP = ""
		nuwa.Status.LoadBalancerIP = ""
		setCondition(nuwa, appv1.ConditionServiceReady, false,
			"", "",
			"ServiceNotFound", "Service not found")
	}

	// Get PVC status if storage type is PVC
	if nuwa.Spec.Storage != nil && nuwa.Spec.Storage.Type == appv1.StorageTypePVC {
		// Get single PVC created by reconcilePVC
		pvcName := getPVCName(nuwa.Name)
		pvc := &corev1.PersistentVolumeClaim{}
		pvcErr := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: nuwa.Namespace}, pvc)

		if pvcErr == nil {
			nuwa.Status.PVCStatus = string(pvc.Status.Phase)
			pvcBound := pvc.Status.Phase == corev1.ClaimBound
			setCondition(nuwa, appv1.ConditionPVCReady, pvcBound,
				"PVCBound", "PVC is bound",
				"PVCNotBound", fmt.Sprintf("PVC status: %s", pvc.Status.Phase))
		} else if errors.IsNotFound(pvcErr) {
			nuwa.Status.PVCStatus = "NotFound"
			setCondition(nuwa, appv1.ConditionPVCReady, false,
				"", "",
				"PVCNotFound", "PVC not found")
		} else {
			log.Error(pvcErr, "Failed to get PVC")
		}
	} else {
		nuwa.Status.PVCStatus = ""
		// Remove PVC condition if storage is not PVC type
		removeCondition(nuwa, appv1.ConditionPVCReady)
	}

	// Determine phase
	if cloneSetErr == nil {
		if cloneSet.Status.ReadyReplicas == cloneSet.Status.Replicas && cloneSet.Status.Replicas > 0 {
			nuwa.Status.Phase = appv1.NuwaPhaseRunning
		} else if cloneSet.Status.Replicas == 0 {
			nuwa.Status.Phase = appv1.NuwaPhasePending
		} else {
			nuwa.Status.Phase = appv1.NuwaPhasePending
		}
	} else {
		nuwa.Status.Phase = appv1.NuwaPhasePending
	}

	return r.Status().Update(ctx, nuwa)
}

// setCondition sets a condition on the Nuwa resource
func setCondition(nuwa *appv1.Nuwa, condType string, status bool, trueReason, trueMsg, falseReason, falseMsg string) {
	condStatus := metav1.ConditionFalse
	reason := falseReason
	message := falseMsg
	if status {
		condStatus = metav1.ConditionTrue
		reason = trueReason
		message = trueMsg
	}

	condition := metav1.Condition{
		Type:               condType,
		Status:             condStatus,
		ObservedGeneration: nuwa.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	// Find and update existing condition or append new one
	for i, c := range nuwa.Status.Conditions {
		if c.Type == condType {
			if c.Status != condStatus {
				nuwa.Status.Conditions[i] = condition
			}
			return
		}
	}
	nuwa.Status.Conditions = append(nuwa.Status.Conditions, condition)
}

// removeCondition removes a condition from the Nuwa resource
func removeCondition(nuwa *appv1.Nuwa, condType string) {
	conditions := make([]metav1.Condition, 0, len(nuwa.Status.Conditions))
	for _, c := range nuwa.Status.Conditions {
		if c.Type != condType {
			conditions = append(conditions, c)
		}
	}
	nuwa.Status.Conditions = conditions
}

// SetupWithManager sets up the controller with the Manager.
func (r *NuwaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.Nuwa{}).
		Owns(&kruiseappsv1alpha1.CloneSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Named("nuwa").
		Complete(r)
}

func resourcePtr(q resource.Quantity) *resource.Quantity {
	return &q
}

// presetLimits defines resource limits for each preset
var presetLimits = map[appv1.ResourcesPreset]corev1.ResourceList{
	appv1.ResourcesPresetNone:    {},
	appv1.ResourcesPresetNano:    {corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("128Mi")},
	appv1.ResourcesPresetMicro:   {corev1.ResourceCPU: resource.MustParse("250m"), corev1.ResourceMemory: resource.MustParse("256Mi")},
	appv1.ResourcesPresetSmall:   {corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("512Mi")},
	appv1.ResourcesPresetMedium:  {corev1.ResourceCPU: resource.MustParse("1"), corev1.ResourceMemory: resource.MustParse("1Gi")},
	appv1.ResourcesPresetLarge:   {corev1.ResourceCPU: resource.MustParse("2"), corev1.ResourceMemory: resource.MustParse("2Gi")},
	appv1.ResourcesPresetXLarge:  {corev1.ResourceCPU: resource.MustParse("4"), corev1.ResourceMemory: resource.MustParse("4Gi")},
	appv1.ResourcesPresetXXLarge: {corev1.ResourceCPU: resource.MustParse("8"), corev1.ResourceMemory: resource.MustParse("8Gi")},
}

// getResourcesFromPreset returns resource requirements based on preset and overcommit level
func getResourcesFromPreset(preset appv1.ResourcesPreset, overcommit appv1.ResourcesOvercommit) corev1.ResourceRequirements {

	limits, ok := presetLimits[preset]
	if !ok || len(limits) == 0 {
		// Return empty requirements if preset is none or unknown
		return corev1.ResourceRequirements{}
	}

	// Determine divisor based on overcommit level (default to none = no overcommit)
	var divisor int64 = 1
	switch overcommit {
	case appv1.ResourcesOvercommitLow:
		divisor = 2
	case appv1.ResourcesOvercommitHigh:
		divisor = 4
	}

	requests := make(corev1.ResourceList)
	for key, val := range limits {
		q := val.DeepCopy()
		if divisor > 1 {
			q.SetMilli(q.MilliValue() / divisor)
		}
		requests[key] = q
	}

	return corev1.ResourceRequirements{
		Requests: requests,
		Limits:   limits,
	}
}

// buildLabels returns standard labels for Nuwa resources
func buildLabels(nuwaName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       nuwaName,
		"app.kubernetes.io/managed-by": "nuwa",
	}
}

// getPVCName returns the PVC name for a Nuwa resource
func getPVCName(nuwaName string) string {
	return nuwaName + "-data"
}
