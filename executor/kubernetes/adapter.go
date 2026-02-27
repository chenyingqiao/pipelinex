package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"github.com/chenyingqiao/pipelinex"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// KubernetesAdapter Kubernetes适配器实现
type KubernetesAdapter struct {
	config map[string]any
}

// NewKubernetesAdapter 创建新的Kubernetes适配器
func NewKubernetesAdapter() *KubernetesAdapter {
	return &KubernetesAdapter{
		config: make(map[string]any),
	}
}

// Config 配置适配器
// 支持的配置项：
//   - namespace: 命名空间（默认：default）
//   - image: 默认镜像
//   - workdir: 工作目录
//   - env: 环境变量 map[string]string
//   - serviceAccount: ServiceAccount名称
//   - configMaps: ConfigMap挂载列表 []map[string]string{"name": "cm-name", "mountPath": "/path"}
//   - secrets: Secret挂载列表 []map[string]string{"name": "secret-name", "mountPath": "/path"}
//   - volumes: PVC挂载列表 []map[string]string{"name": "pvc-name", "mountPath": "/path"}
//   - emptyDirs: EmptyDir挂载列表 []map[string]string{"name": "tmp", "mountPath": "/tmp"}
//   - tty: 是否启用TTY模式 bool
//   - ttyWidth: TTY终端宽度 int（默认80）
//   - ttyHeight: TTY终端高度 int（默认24）
//   - podReadyTimeout: Pod就绪等待超时时间（秒，默认60）
func (a *KubernetesAdapter) Config(ctx context.Context, config map[string]any) error {
	a.config = config
	return nil
}

// 确保KubernetesAdapter实现了Adapter接口
var _ pipelinex.Adapter = (*KubernetesAdapter)(nil)

// parseVolumeMount 解析卷挂载配置
func parseVolumeMount(volume map[string]any) (name, mountPath string, err error) {
	nameVal, ok := volume["name"]
	if !ok {
		return "", "", fmt.Errorf("volume must have a 'name' field")
	}
	name, ok = nameVal.(string)
	if !ok || name == "" {
		return "", "", fmt.Errorf("volume name must be a non-empty string")
	}

	pathVal, ok := volume["mountPath"]
	if !ok {
		return "", "", fmt.Errorf("volume must have a 'mountPath' field")
	}
	mountPath, ok = pathVal.(string)
	if !ok || mountPath == "" {
		return "", "", fmt.Errorf("volume mountPath must be a non-empty string")
	}

	return name, mountPath, nil
}

// parseConfigMapVolume 解析ConfigMap卷配置
func parseConfigMapVolume(volume map[string]any) (corev1.Volume, corev1.VolumeMount, error) {
	name, mountPath, err := parseVolumeMount(volume)
	if err != nil {
		return corev1.Volume{}, corev1.VolumeMount{}, err
	}

	// 获取ConfigMap名称（可选，默认使用volume name）
	cmName := name
	if cmNameVal, ok := volume["configMapName"]; ok {
		if cmNameStr, ok := cmNameVal.(string); ok && cmNameStr != "" {
			cmName = cmNameStr
		}
	}

	// 获取是否可选
	optional := false
	if optionalVal, ok := volume["optional"]; ok {
		if optionalBool, ok := optionalVal.(bool); ok {
			optional = optionalBool
		}
	}

	volumeConfig := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: cmName,
				},
				Optional: &optional,
			},
		},
	}

	mount := corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	}

	// 处理子路径
	if subPath, ok := volume["subPath"]; ok {
		if subPathStr, ok := subPath.(string); ok {
			mount.SubPath = subPathStr
		}
	}

	return volumeConfig, mount, nil
}

// parseSecretVolume 解析Secret卷配置
func parseSecretVolume(volume map[string]any) (corev1.Volume, corev1.VolumeMount, error) {
	name, mountPath, err := parseVolumeMount(volume)
	if err != nil {
		return corev1.Volume{}, corev1.VolumeMount{}, err
	}

	// 获取Secret名称（可选，默认使用volume name）
	secretName := name
	if secretNameVal, ok := volume["secretName"]; ok {
		if secretNameStr, ok := secretNameVal.(string); ok && secretNameStr != "" {
			secretName = secretNameStr
		}
	}

	// 获取是否可选
	optional := false
	if optionalVal, ok := volume["optional"]; ok {
		if optionalBool, ok := optionalVal.(bool); ok {
			optional = optionalBool
		}
	}

	volumeConfig := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
				Optional:   &optional,
			},
		},
	}

	mount := corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	}

	// 处理子路径
	if subPath, ok := volume["subPath"]; ok {
		if subPathStr, ok := subPath.(string); ok {
			mount.SubPath = subPathStr
		}
	}

	return volumeConfig, mount, nil
}

// parsePVCVolume 解析PVC卷配置
func parsePVCVolume(volume map[string]any) (corev1.Volume, corev1.VolumeMount, error) {
	name, mountPath, err := parseVolumeMount(volume)
	if err != nil {
		return corev1.Volume{}, corev1.VolumeMount{}, err
	}

	// 获取PVC名称（可选，默认使用volume name）
	pvcName := name
	if pvcNameVal, ok := volume["claimName"]; ok {
		if pvcNameStr, ok := pvcNameVal.(string); ok && pvcNameStr != "" {
			pvcName = pvcNameStr
		}
	}

	volumeConfig := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
			},
		},
	}

	mount := corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	}

	// 处理只读
	if readOnly, ok := volume["readOnly"]; ok {
		if readOnlyBool, ok := readOnly.(bool); ok {
			mount.ReadOnly = readOnlyBool
		}
	}

	// 处理子路径
	if subPath, ok := volume["subPath"]; ok {
		if subPathStr, ok := subPath.(string); ok {
			mount.SubPath = subPathStr
		}
	}

	return volumeConfig, mount, nil
}

// parseEmptyDirVolume 解析EmptyDir卷配置
func parseEmptyDirVolume(volume map[string]any) (corev1.Volume, corev1.VolumeMount, error) {
	name, mountPath, err := parseVolumeMount(volume)
	if err != nil {
		return corev1.Volume{}, corev1.VolumeMount{}, err
	}

	volumeConfig := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	mount := corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	}

	// 处理大小限制
	if sizeLimit, ok := volume["sizeLimit"]; ok {
		if sizeLimitStr, ok := sizeLimit.(string); ok {
			// 解析大小限制（例如："1Gi"）
			quantity := parseQuantity(sizeLimitStr)
			volumeConfig.VolumeSource.EmptyDir.SizeLimit = &quantity
		}
	}

	// 处理介质
	if medium, ok := volume["medium"]; ok {
		if mediumStr, ok := medium.(string); ok {
			switch strings.ToLower(mediumStr) {
			case "memory":
				volumeConfig.VolumeSource.EmptyDir.Medium = corev1.StorageMediumMemory
			default:
				volumeConfig.VolumeSource.EmptyDir.Medium = corev1.StorageMediumDefault
			}
		}
	}

	return volumeConfig, mount, nil
}

// parseQuantity 解析资源数量
func parseQuantity(s string) resource.Quantity {
	return resource.MustParse(s)
}
