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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kruiseappsv1alpha1 "github.com/openkruise/kruise-api/apps/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appv1 "github.com/ysicing/nuwa/api/v1"
)

var _ = Describe("Nuwa Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		nuwa := &appv1.Nuwa{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind Nuwa")
			err := k8sClient.Get(ctx, typeNamespacedName, nuwa)
			if err != nil && errors.IsNotFound(err) {
				resource := &appv1.Nuwa{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &appv1.Nuwa{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Nuwa")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &NuwaReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})

var _ = Describe("buildUpdateStrategy", func() {
	It("should return default strategy when input is nil", func() {
		strategy := buildUpdateStrategy(nil)
		Expect(strategy.Type).To(Equal(kruiseappsv1alpha1.InPlaceIfPossibleCloneSetUpdateStrategyType))
		Expect(strategy.Partition).To(BeNil())
		Expect(strategy.MaxUnavailable).To(BeNil())
		Expect(strategy.MaxSurge).To(BeNil())
	})

	It("should map InPlaceIfPossible strategy correctly", func() {
		input := &appv1.UpdateStrategy{Type: appv1.InPlaceIfPossibleUpdateStrategy}
		strategy := buildUpdateStrategy(input)
		Expect(strategy.Type).To(Equal(kruiseappsv1alpha1.InPlaceIfPossibleCloneSetUpdateStrategyType))
	})

	It("should map ReCreate strategy correctly", func() {
		input := &appv1.UpdateStrategy{Type: appv1.ReCreateUpdateStrategy}
		strategy := buildUpdateStrategy(input)
		Expect(strategy.Type).To(Equal(kruiseappsv1alpha1.RecreateCloneSetUpdateStrategyType))
	})

	It("should map InPlaceOnly strategy correctly", func() {
		input := &appv1.UpdateStrategy{Type: appv1.InPlaceOnlyUpdateStrategy}
		strategy := buildUpdateStrategy(input)
		Expect(strategy.Type).To(Equal(kruiseappsv1alpha1.InPlaceOnlyCloneSetUpdateStrategyType))
	})

	It("should set partition when provided", func() {
		partition := int32(2)
		input := &appv1.UpdateStrategy{Partition: &partition}
		strategy := buildUpdateStrategy(input)
		Expect(strategy.Partition).NotTo(BeNil())
		Expect(strategy.Partition.IntVal).To(Equal(partition))
	})

	It("should set maxUnavailable when provided", func() {
		maxUnavailable := intstr.FromInt(3)
		input := &appv1.UpdateStrategy{MaxUnavailable: &maxUnavailable}
		strategy := buildUpdateStrategy(input)
		Expect(strategy.MaxUnavailable).NotTo(BeNil())
		Expect(strategy.MaxUnavailable.IntVal).To(Equal(int32(3)))
	})

	It("should set maxSurge when provided", func() {
		maxSurge := intstr.FromInt(1)
		input := &appv1.UpdateStrategy{MaxSurge: &maxSurge}
		strategy := buildUpdateStrategy(input)
		Expect(strategy.MaxSurge).NotTo(BeNil())
		Expect(strategy.MaxSurge.IntVal).To(Equal(int32(1)))
	})

	It("should set gracePeriodSeconds when provided", func() {
		gracePeriod := int32(60)
		input := &appv1.UpdateStrategy{GracePeriodSeconds: &gracePeriod}
		strategy := buildUpdateStrategy(input)
		Expect(strategy.InPlaceUpdateStrategy).NotTo(BeNil())
		Expect(strategy.InPlaceUpdateStrategy.GracePeriodSeconds).To(Equal(gracePeriod))
	})

	It("should handle all fields together", func() {
		partition := int32(5)
		maxUnavailable := intstr.FromString("30%")
		maxSurge := intstr.FromInt(2)
		gracePeriod := int32(90)
		input := &appv1.UpdateStrategy{
			Type:               appv1.InPlaceOnlyUpdateStrategy,
			Partition:          &partition,
			MaxUnavailable:     &maxUnavailable,
			MaxSurge:           &maxSurge,
			GracePeriodSeconds: &gracePeriod,
		}
		strategy := buildUpdateStrategy(input)
		Expect(strategy.Type).To(Equal(kruiseappsv1alpha1.InPlaceOnlyCloneSetUpdateStrategyType))
		Expect(strategy.Partition.IntVal).To(Equal(partition))
		Expect(strategy.MaxUnavailable.StrVal).To(Equal("30%"))
		Expect(strategy.MaxSurge.IntVal).To(Equal(int32(2)))
		Expect(strategy.InPlaceUpdateStrategy.GracePeriodSeconds).To(Equal(gracePeriod))
	})
})

var _ = Describe("getResourcesFromPreset", func() {
	Context("when preset is none", func() {
		It("should return empty resources", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetNone, appv1.ResourcesOvercommitNone)
			Expect(resources.Limits).To(BeEmpty())
			Expect(resources.Requests).To(BeEmpty())
		})

		It("should return empty resources regardless of overcommit", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetNone, appv1.ResourcesOvercommitLow)
			Expect(resources.Limits).To(BeEmpty())
			Expect(resources.Requests).To(BeEmpty())
		})
	})

	Context("when preset is nano", func() {
		It("should return correct resources with no overcommit", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetNano, appv1.ResourcesOvercommitNone)
			Expect(resources.Limits.Cpu().String()).To(Equal("100m"))
			Expect(resources.Limits.Memory().String()).To(Equal("128Mi"))
			Expect(resources.Requests.Cpu().String()).To(Equal("100m"))
			Expect(resources.Requests.Memory().String()).To(Equal("128Mi"))
		})

		It("should return correct resources with low overcommit", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetNano, appv1.ResourcesOvercommitLow)
			Expect(resources.Limits.Cpu().String()).To(Equal("100m"))
			Expect(resources.Limits.Memory().String()).To(Equal("128Mi"))
			Expect(resources.Requests.Cpu().String()).To(Equal("50m"))
			Expect(resources.Requests.Memory().Value()).To(Equal(int64(64 * 1024 * 1024)))
		})

		It("should return correct resources with high overcommit", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetNano, appv1.ResourcesOvercommitHigh)
			Expect(resources.Limits.Cpu().String()).To(Equal("100m"))
			Expect(resources.Limits.Memory().String()).To(Equal("128Mi"))
			Expect(resources.Requests.Cpu().String()).To(Equal("25m"))
			Expect(resources.Requests.Memory().Value()).To(Equal(int64(32 * 1024 * 1024)))
		})
	})

	Context("when preset is small", func() {
		It("should return correct resources with no overcommit", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetSmall, appv1.ResourcesOvercommitNone)
			Expect(resources.Limits.Cpu().String()).To(Equal("500m"))
			Expect(resources.Limits.Memory().String()).To(Equal("512Mi"))
			Expect(resources.Requests.Cpu().String()).To(Equal("500m"))
			Expect(resources.Requests.Memory().String()).To(Equal("512Mi"))
		})

		It("should return correct resources with low overcommit", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetSmall, appv1.ResourcesOvercommitLow)
			Expect(resources.Limits.Cpu().String()).To(Equal("500m"))
			Expect(resources.Limits.Memory().String()).To(Equal("512Mi"))
			Expect(resources.Requests.Cpu().String()).To(Equal("250m"))
			Expect(resources.Requests.Memory().Value()).To(Equal(int64(256 * 1024 * 1024)))
		})

		It("should return correct resources with high overcommit", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetSmall, appv1.ResourcesOvercommitHigh)
			Expect(resources.Limits.Cpu().String()).To(Equal("500m"))
			Expect(resources.Limits.Memory().String()).To(Equal("512Mi"))
			Expect(resources.Requests.Cpu().String()).To(Equal("125m"))
			Expect(resources.Requests.Memory().Value()).To(Equal(int64(128 * 1024 * 1024)))
		})
	})

	Context("when preset is medium", func() {
		It("should return correct resources with no overcommit", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetMedium, appv1.ResourcesOvercommitNone)
			Expect(resources.Limits.Cpu().String()).To(Equal("1"))
			Expect(resources.Limits.Memory().String()).To(Equal("1Gi"))
			Expect(resources.Requests.Cpu().String()).To(Equal("1"))
			Expect(resources.Requests.Memory().String()).To(Equal("1Gi"))
		})

		It("should return correct resources with low overcommit", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetMedium, appv1.ResourcesOvercommitLow)
			Expect(resources.Limits.Cpu().String()).To(Equal("1"))
			Expect(resources.Limits.Memory().String()).To(Equal("1Gi"))
			Expect(resources.Requests.Cpu().String()).To(Equal("500m"))
			Expect(resources.Requests.Memory().Value()).To(Equal(int64(512 * 1024 * 1024)))
		})
	})

	Context("when preset is large", func() {
		It("should return correct resources with no overcommit", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetLarge, appv1.ResourcesOvercommitNone)
			Expect(resources.Limits.Cpu().String()).To(Equal("2"))
			Expect(resources.Limits.Memory().String()).To(Equal("2Gi"))
			Expect(resources.Requests.Cpu().String()).To(Equal("2"))
			Expect(resources.Requests.Memory().String()).To(Equal("2Gi"))
		})

		It("should return correct resources with low overcommit", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetLarge, appv1.ResourcesOvercommitLow)
			Expect(resources.Limits.Cpu().String()).To(Equal("2"))
			Expect(resources.Limits.Memory().String()).To(Equal("2Gi"))
			Expect(resources.Requests.Cpu().String()).To(Equal("1"))
			Expect(resources.Requests.Memory().Value()).To(Equal(int64(1024 * 1024 * 1024)))
		})
	})

	Context("when preset is xlarge", func() {
		It("should return correct resources with high overcommit", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetXLarge, appv1.ResourcesOvercommitHigh)
			Expect(resources.Limits.Cpu().String()).To(Equal("4"))
			Expect(resources.Limits.Memory().String()).To(Equal("4Gi"))
			Expect(resources.Requests.Cpu().String()).To(Equal("1"))
			Expect(resources.Requests.Memory().Value()).To(Equal(int64(1024 * 1024 * 1024)))
		})
	})

	Context("when preset is xxlarge", func() {
		It("should return correct resources with low overcommit", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetXXLarge, appv1.ResourcesOvercommitLow)
			Expect(resources.Limits.Cpu().String()).To(Equal("8"))
			Expect(resources.Limits.Memory().String()).To(Equal("8Gi"))
			Expect(resources.Requests.Cpu().String()).To(Equal("4"))
			Expect(resources.Requests.Memory().Value()).To(Equal(int64(4 * 1024 * 1024 * 1024)))
		})
	})

	Context("edge cases", func() {
		It("should handle empty overcommit (defaults to none)", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetSmall, "")
			Expect(resources.Limits.Cpu().String()).To(Equal("500m"))
			Expect(resources.Requests.Cpu().String()).To(Equal("500m"))
		})

		It("should preserve precision for millicore values", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetMicro, appv1.ResourcesOvercommitLow)
			// 250m / 2 = 125m (should not round to 0)
			cpuRequest := resources.Requests.Cpu()
			Expect(cpuRequest.MilliValue()).To(Equal(int64(125)))
		})

		It("should preserve precision for memory values", func() {
			resources := getResourcesFromPreset(appv1.ResourcesPresetNano, appv1.ResourcesOvercommitHigh)
			// 128Mi / 4 = 32Mi
			memRequest := resources.Requests.Memory()
			expectedBytes := int64(32 * 1024 * 1024)
			Expect(memRequest.Value()).To(Equal(expectedBytes))
		})
	})
})

var _ = Describe("resourcePtr", func() {
	It("should return a pointer to the quantity", func() {
		q := resource.MustParse("1Gi")
		ptr := resourcePtr(q)
		Expect(ptr).NotTo(BeNil())
		Expect(ptr.String()).To(Equal("1Gi"))
	})
})

var _ = Describe("getRetainPolicy", func() {
	It("should return Retain when storage is nil", func() {
		policy := getRetainPolicy(nil)
		Expect(policy).To(Equal(appv1.PVCRetainPolicyRetain))
	})

	It("should return Retain when retainPolicy is empty", func() {
		storage := &appv1.StorageSpec{}
		policy := getRetainPolicy(storage)
		Expect(policy).To(Equal(appv1.PVCRetainPolicyRetain))
	})

	It("should return Retain when explicitly set to Retain", func() {
		storage := &appv1.StorageSpec{
			RetainPolicy: appv1.PVCRetainPolicyRetain,
		}
		policy := getRetainPolicy(storage)
		Expect(policy).To(Equal(appv1.PVCRetainPolicyRetain))
	})

	It("should return Delete when explicitly set to Delete", func() {
		storage := &appv1.StorageSpec{
			RetainPolicy: appv1.PVCRetainPolicyDelete,
		}
		policy := getRetainPolicy(storage)
		Expect(policy).To(Equal(appv1.PVCRetainPolicyDelete))
	})
})

var _ = Describe("hasNuwaOwnerReference", func() {
	var pvc *corev1.PersistentVolumeClaim
	var nuwa *appv1.Nuwa

	BeforeEach(func() {
		pvc = &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pvc",
				Namespace: "default",
			},
		}
		nuwa = &appv1.Nuwa{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-nuwa",
				Namespace: "default",
			},
		}
	})

	It("should return false when PVC has no owner references", func() {
		result := hasNuwaOwnerReference(pvc, nuwa)
		Expect(result).To(BeFalse())
	})

	It("should return false when PVC has different owner reference", func() {
		pvc.OwnerReferences = []metav1.OwnerReference{
			{
				Kind: "Deployment",
				Name: "other-owner",
			},
		}
		result := hasNuwaOwnerReference(pvc, nuwa)
		Expect(result).To(BeFalse())
	})

	It("should return false when PVC has Nuwa owner with different name", func() {
		pvc.OwnerReferences = []metav1.OwnerReference{
			{
				Kind: "Nuwa",
				Name: "different-nuwa",
			},
		}
		result := hasNuwaOwnerReference(pvc, nuwa)
		Expect(result).To(BeFalse())
	})

	It("should return true when PVC has matching Nuwa owner reference", func() {
		pvc.OwnerReferences = []metav1.OwnerReference{
			{
				Kind: "Nuwa",
				Name: "test-nuwa",
			},
		}
		result := hasNuwaOwnerReference(pvc, nuwa)
		Expect(result).To(BeTrue())
	})

	It("should return true when PVC has multiple owner references including Nuwa", func() {
		pvc.OwnerReferences = []metav1.OwnerReference{
			{
				Kind: "Deployment",
				Name: "other-owner",
			},
			{
				Kind: "Nuwa",
				Name: "test-nuwa",
			},
		}
		result := hasNuwaOwnerReference(pvc, nuwa)
		Expect(result).To(BeTrue())
	})
})

var _ = Describe("removeNuwaOwnerReference", func() {
	var pvc *corev1.PersistentVolumeClaim
	var nuwa *appv1.Nuwa

	BeforeEach(func() {
		pvc = &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pvc",
				Namespace: "default",
			},
		}
		nuwa = &appv1.Nuwa{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-nuwa",
				Namespace: "default",
			},
		}
	})

	It("should do nothing when PVC has no owner references", func() {
		removeNuwaOwnerReference(pvc, nuwa)
		Expect(pvc.OwnerReferences).To(BeNil())
	})

	It("should remove Nuwa owner reference", func() {
		pvc.OwnerReferences = []metav1.OwnerReference{
			{
				Kind: "Nuwa",
				Name: "test-nuwa",
			},
		}
		removeNuwaOwnerReference(pvc, nuwa)
		Expect(pvc.OwnerReferences).To(BeEmpty())
	})

	It("should preserve other owner references", func() {
		pvc.OwnerReferences = []metav1.OwnerReference{
			{
				Kind: "Deployment",
				Name: "other-owner",
			},
			{
				Kind: "Nuwa",
				Name: "test-nuwa",
			},
			{
				Kind: "StatefulSet",
				Name: "another-owner",
			},
		}
		removeNuwaOwnerReference(pvc, nuwa)
		Expect(pvc.OwnerReferences).To(HaveLen(2))
		Expect(pvc.OwnerReferences[0].Kind).To(Equal("Deployment"))
		Expect(pvc.OwnerReferences[1].Kind).To(Equal("StatefulSet"))
	})

	It("should not remove Nuwa owner reference with different name", func() {
		pvc.OwnerReferences = []metav1.OwnerReference{
			{
				Kind: "Nuwa",
				Name: "different-nuwa",
			},
		}
		removeNuwaOwnerReference(pvc, nuwa)
		Expect(pvc.OwnerReferences).To(HaveLen(1))
		Expect(pvc.OwnerReferences[0].Name).To(Equal("different-nuwa"))
	})
})

var _ = Describe("getPVCName", func() {
	It("should return correct PVC name", func() {
		name := getPVCName("my-app")
		Expect(name).To(Equal("my-app-data"))
	})

	It("should handle empty name", func() {
		name := getPVCName("")
		Expect(name).To(Equal("-data"))
	})
})

var _ = Describe("buildLabels", func() {
	It("should return correct labels", func() {
		labels := buildLabels("my-app")
		Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/name", "my-app"))
		Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "nuwa"))
	})

	It("should handle empty name", func() {
		labels := buildLabels("")
		Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/name", ""))
		Expect(labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "nuwa"))
	})
})
