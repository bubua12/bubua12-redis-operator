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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "github.com/bubua12/bubua12-redis-operator/api/v1alpha1"
)

// RedisReconciler reconciles a Redis object
type RedisReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cache.bubua12.com,resources=redis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.bubua12.com,resources=redis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.bubua12.com,resources=redis/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Redis object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.1/pkg/reconcile
func (r *RedisReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// 1. 获取 Redis 对象
	redis := &cachev1alpha1.Redis{}
	if err := r.Get(ctx, req.NamespacedName, redis); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Redis 资源已被删除，跳过")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("开始处理 Redis", "name", redis.Name, "replicas", *redis.Spec.Replicas)

	// 2. 确保 StatefulSet 存在且正确
	if err := r.ensureStatefulSet(ctx, redis); err != nil {
		log.Error(err, "创建/更新 StatefulSet 失败")
		return ctrl.Result{}, err
	}

	// 3. 确保 Service 存在
	if err := r.ensureService(ctx, redis); err != nil {
		log.Error(err, "创建/更新 Service 失败")
		return ctrl.Result{}, err
	}

	// 4. 更新 Status
	if err := r.updateStatus(ctx, redis); err != nil {
		log.Error(err, "更新 Status 失败")
		return ctrl.Result{}, err
	}

	log.Info("Redis 处理完成", "name", redis.Name)
	return ctrl.Result{}, nil
}

// ensureStatefulSet 确保 StatefulSet 存在且配置正确（Upsert 模式）
func (r *RedisReconciler) ensureStatefulSet(ctx context.Context, redis *cachev1alpha1.Redis) error {
	log := logf.FromContext(ctx)
	desired := r.buildStatefulSet(redis)

	// 查找是否已存在
	existing := &appsv1.StatefulSet{}
	err := r.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if errors.IsNotFound(err) {
		// 不存在 → 创建
		log.Info("创建 StatefulSet", "name", desired.Name)
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// 已存在 → 更新 replicas 和 image
	existing.Spec.Replicas = desired.Spec.Replicas
	existing.Spec.Template.Spec.Containers[0].Image = desired.Spec.Template.Spec.Containers[0].Image
	log.Info("更新 StatefulSet", "name", existing.Name)
	return r.Update(ctx, existing)
}

// buildStatefulSet 构建期望的 StatefulSet 对象
func (r *RedisReconciler) buildStatefulSet(redis *cachev1alpha1.Redis) *appsv1.StatefulSet {
	labels := map[string]string{
		"app":     "redis",
		"managed": "redis-operator",
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      redis.Name,
			Namespace: redis.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: redis.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			ServiceName: redis.Name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "redis",
							Image: redis.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "redis",
									ContainerPort: redis.Spec.Port,
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
						},
					},
				},
			},
		},
	}
}

// ensureService 确保 Service 存在
func (r *RedisReconciler) ensureService(ctx context.Context, redis *cachev1alpha1.Redis) error {
	log := logf.FromContext(ctx)

	labels := map[string]string{
		"app":     "redis",
		"managed": "redis-operator",
	}

	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      redis.Name,
			Namespace: redis.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "redis",
					Port:       redis.Spec.Port,
					TargetPort: intstr.FromString("redis"),
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	existing := &corev1.Service{}
	err := r.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if errors.IsNotFound(err) {
		log.Info("创建 Service", "name", desired.Name)
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Service 已存在，只更新端口配置
	existing.Spec.Ports = desired.Spec.Ports
	return r.Update(ctx, existing)
}

// updateStatus 读取 StatefulSet 的实际状态，更新到 Redis.Status
func (r *RedisReconciler) updateStatus(ctx context.Context, redis *cachev1alpha1.Redis) error {
	log := logf.FromContext(ctx)

	// 读取 StatefulSet 的实际状态
	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, client.ObjectKey{Name: redis.Name, Namespace: redis.Namespace}, sts); err != nil {
		if errors.IsNotFound(err) {
			// StatefulSet 还没创建完，状态设为 Creating
			redis.Status.State = "Creating"
			redis.Status.ReadyReplicas = 0
			return r.Status().Update(ctx, redis)
		}
		return err
	}

	// 对比期望和实际，决定状态
	redis.Status.ReadyReplicas = sts.Status.ReadyReplicas

	if sts.Status.ReadyReplicas == *redis.Spec.Replicas {
		redis.Status.State = "Running"
	} else if sts.Status.ReadyReplicas > 0 {
		redis.Status.State = "Progressing"
	} else {
		redis.Status.State = "Creating"
	}

	log.Info("更新 Status",
		"readyReplicas", redis.Status.ReadyReplicas,
		"state", redis.Status.State,
	)
	return r.Status().Update(ctx, redis)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Redis{}).
		Named("redis").
		Complete(r)
}
