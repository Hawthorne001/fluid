/*
  Copyright 2022 The Fluid Authors.

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

package thin

import (
	"reflect"
	"strings"
	"testing"

	datav1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	"github.com/fluid-cloudnative/fluid/pkg/common"
	"github.com/fluid-cloudnative/fluid/pkg/ddc/base"
	"github.com/fluid-cloudnative/fluid/pkg/utils/fake"
	"github.com/fluid-cloudnative/fluid/pkg/utils/testutil"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestThinEngine_parseFromProfileFuse(t1 *testing.T) {
	profile := datav1alpha1.ThinRuntimeProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "fluid",
		},
		Spec: datav1alpha1.ThinRuntimeProfileSpec{
			Fuse: datav1alpha1.ThinFuseSpec{
				Image:           "test",
				ImageTag:        "v1",
				ImagePullPolicy: "Always",
				Env: []corev1.EnvVar{{
					Name:  "a",
					Value: "b",
				}, {
					Name: "b",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "test-cm",
							},
						},
					},
				}},
				NodeSelector: map[string]string{"a": "b"},
				Ports: []corev1.ContainerPort{{
					Name:          "port",
					ContainerPort: 8080,
				}},
				LivenessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/healthz",
						},
					},
					InitialDelaySeconds: 1,
					TimeoutSeconds:      1,
					PeriodSeconds:       1,
					SuccessThreshold:    1,
					FailureThreshold:    1,
				},
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/healthz",
						},
					},
					InitialDelaySeconds: 1,
					TimeoutSeconds:      1,
					PeriodSeconds:       1,
					SuccessThreshold:    1,
					FailureThreshold:    1,
				},
				NetworkMode: datav1alpha1.HostNetworkMode,
			},
		},
	}
	wantValue := &ThinValue{
		Fuse: Fuse{
			Image:           "test",
			ImageTag:        "v1",
			ImagePullPolicy: "Always",
			HostNetwork:     true,
			Envs: []corev1.EnvVar{{
				Name:  "a",
				Value: "b",
			}, {
				Name: "b",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-cm",
						},
					},
				},
			}},
			Resources: common.Resources{
				Requests: map[corev1.ResourceName]string{},
				Limits:   map[corev1.ResourceName]string{},
			},
			NodeSelector: map[string]string{"a": "b"},
			Ports: []corev1.ContainerPort{{
				Name:          "port",
				ContainerPort: 8080,
			}},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
					},
				},
				InitialDelaySeconds: 1,
				TimeoutSeconds:      1,
				PeriodSeconds:       1,
				SuccessThreshold:    1,
				FailureThreshold:    1,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
					},
				},
				InitialDelaySeconds: 1,
				TimeoutSeconds:      1,
				PeriodSeconds:       1,
				SuccessThreshold:    1,
				FailureThreshold:    1,
			},
		},
	}
	value := &ThinValue{}
	t1.Run("test", func(t1 *testing.T) {
		t := &ThinEngine{
			Log: fake.NullLogger(),
		}
		t.parseFromProfileFuse(&profile, value)
		if !reflect.DeepEqual(value.Fuse, wantValue.Fuse) {
			t1.Errorf("parseFromProfileFuse() got = %v, want = %v", value, wantValue)
		}
	})
}

func TestThinEngine_parseFuseImage(t1 *testing.T) {
	type args struct {
		runtime *datav1alpha1.ThinRuntime
		value   *ThinValue
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test",
			args: args{
				runtime: &datav1alpha1.ThinRuntime{
					Spec: datav1alpha1.ThinRuntimeSpec{
						Fuse: datav1alpha1.ThinFuseSpec{
							Image:           "test",
							ImageTag:        "v1",
							ImagePullPolicy: "Always",
						},
					},
				},
				value: &ThinValue{},
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &ThinEngine{}
			t.parseFuseImage(tt.args.runtime, tt.args.value)
			if tt.args.value.Fuse.Image != tt.args.runtime.Spec.Fuse.Image ||
				tt.args.value.Fuse.ImageTag != tt.args.runtime.Spec.Fuse.ImageTag ||
				tt.args.value.Fuse.ImagePullPolicy != tt.args.runtime.Spec.Fuse.ImagePullPolicy {
				t1.Errorf("got %v, want %v", tt.args.value.Fuse, tt.args.runtime.Spec.Fuse)
			}
		})
	}
}

func TestThinEngine_parseFuseOptions(t1 *testing.T) {
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sec",
			Namespace: "fluid",
		},
		Data: map[string][]byte{
			"a": []byte("z"),
			"h": []byte("i"),
		},
	}
	testObjs := []runtime.Object{}
	testObjs = append(testObjs, (*sec).DeepCopy())

	client := fake.NewFakeClientWithScheme(testScheme, testObjs...)
	type args struct {
		runtime *datav1alpha1.ThinRuntime
		profile *datav1alpha1.ThinRuntimeProfile
		dataset *datav1alpha1.Dataset
	}
	tests := []struct {
		name       string
		args       args
		wantOption map[string]string
	}{
		{
			name: "test",
			args: args{
				runtime: &datav1alpha1.ThinRuntime{
					Spec: datav1alpha1.ThinRuntimeSpec{Fuse: datav1alpha1.ThinFuseSpec{Options: map[string]string{
						"a": "x",
						"c": "x",
					}}},
				},
				profile: &datav1alpha1.ThinRuntimeProfile{
					Spec: datav1alpha1.ThinRuntimeProfileSpec{Fuse: datav1alpha1.ThinFuseSpec{Options: map[string]string{
						"a": "y",
						"b": "y",
					}}},
				},
				dataset: &datav1alpha1.Dataset{Spec: datav1alpha1.DatasetSpec{
					SharedOptions: map[string]string{
						"f": "g",
						"e": "e",
					},
					Mounts: []datav1alpha1.Mount{{
						Options: map[string]string{
							"d": "z",
							"e": "",
						},
					},
					}}},
			},
			wantOption: map[string]string{
				"a": "x",
				"b": "y",
				"c": "x",
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := ThinEngine{
				Client:    client,
				Log:       fake.NullLogger(),
				namespace: "fluid",
			}
			gotOption, err := t.parseFuseOptions(tt.args.runtime, tt.args.profile, tt.args.dataset)
			if err != nil {
				t1.Errorf("parseFuseOptions() err = %v", err)
			}
			options := strings.Split(gotOption, ",")
			if len(options) != len(tt.wantOption) {
				t1.Errorf("parseFuseOptions() got = %v, want = %v", gotOption, tt.wantOption)
			}
			for _, option := range options {
				o := strings.Split(option, "=")
				if len(o) == 1 && tt.wantOption[o[0]] != "" {
					t1.Errorf("parseFuseOptions() got = %v, want = %v", gotOption, tt.wantOption)
				}
				if len(o) == 2 && tt.wantOption[o[0]] != o[1] {
					t1.Errorf("parseFuseOptions() got = %v, want = %v", gotOption, tt.wantOption)
				}
			}
		})
	}
}

func TestThinEngine_transformFuse(t1 *testing.T) {
	profile := buildBaseThinRuntimeProfile()
	runtime := buildBaseThinRuntime()
	dataset := buildBaseDataset()
	wantValue := &ThinValue{
		Fuse: Fuse{
			Enabled:         true,
			Image:           "test",
			ImageTag:        "v1",
			ImagePullPolicy: "Always",
			TargetPath:      "/thin/fluid/test/thin-fuse",
			Resources: common.Resources{
				Requests: map[corev1.ResourceName]string{
					corev1.ResourceCPU:    "100m",
					corev1.ResourceMemory: "1Gi",
				},
				Limits: map[corev1.ResourceName]string{
					corev1.ResourceCPU:    "200m",
					corev1.ResourceMemory: "4Gi",
				},
			},
			HostNetwork: true,
			Envs: []corev1.EnvVar{{
				Name:  "a",
				Value: "b",
			}, {
				Name: "b",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-cm",
						},
					},
				},
			}, {
				Name:  common.ThinFuseOptionEnvKey,
				Value: "fuse-opt=foo",
			}, {
				Name:  common.ThinFusePointEnvKey,
				Value: "/thin/fluid/test/thin-fuse",
			}},
			NodeSelector: map[string]string{"b": "c", "fluid.io/f-fluid-test": "true"},
			Ports: []corev1.ContainerPort{{
				Name:          "port",
				ContainerPort: 8080,
			}},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
					},
				},
				InitialDelaySeconds: 1,
				TimeoutSeconds:      1,
				PeriodSeconds:       1,
				SuccessThreshold:    1,
				FailureThreshold:    1,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
					},
				},
				InitialDelaySeconds: 1,
				TimeoutSeconds:      1,
				PeriodSeconds:       1,
				SuccessThreshold:    1,
				FailureThreshold:    1,
			},
			Volumes: []corev1.Volume{{
				Name: "a",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/test"},
				},
			}, {
				Name: "b",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/b"},
				},
			}},
			VolumeMounts: []corev1.VolumeMount{{
				Name:      "a",
				MountPath: "/test",
			}, {
				Name:      "b",
				MountPath: "/b",
			}},
			// ConfigValue: "{\"/thin/fluid/test/thin-fuse\":\"a=b\"}",
			// MountPath:   "/thin/fluid/test/thin-fuse",
			ConfigValue:   "{\"mounts\":[{\"mountPoint\":\"abc\",\"options\":{\"a\":\"b\",\"c\":\"d\"}}],\"targetPath\":\"/thin/fluid/test/thin-fuse\",\"runtimeOptions\":{\"fuse-opt\":\"foo\"},\"accessModes\":[\"ReadWriteMany\"]}",
			ConfigStorage: "configmap",
			Lifecycle: &corev1.Lifecycle{
				PreStop: &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"umount",
							"/thin/fluid/test/thin-fuse",
						},
					},
				},
			},
		},
	}
	value := &ThinValue{}
	runtimeInfo, err := base.BuildRuntimeInfo("test", "fluid", "thin")
	if err != nil {
		t1.Errorf("fail to create the runtimeInfo with error %v", err)
	}
	t1.Run("test", func(t1 *testing.T) {
		t := &ThinEngine{
			Log:         fake.NullLogger(),
			namespace:   "fluid",
			name:        "test",
			runtime:     runtime,
			runtimeInfo: runtimeInfo,
			Client:      fake.NewFakeClientWithScheme(testScheme),
		}
		if err := t.transformFuse(runtime, profile, dataset, value); err != nil {
			t1.Errorf("transformFuse() error = %v", err)
		}

		value.Fuse.Envs = testutil.SortEnvVarByName(value.Fuse.Envs, common.ThinFuseOptionEnvKey)
		if !testutil.DeepEqualIgnoringSliceOrder(t1, value.Fuse, wantValue.Fuse) {
			valueYaml, _ := yaml.Marshal(value.Fuse)
			wantYaml, _ := yaml.Marshal(wantValue.Fuse)
			t1.Errorf("transformFuse() \ngot = %v, \nwant = %v", string(valueYaml), string(wantYaml))
		}
	})
}

func TestThinEngine_transformFuseWithCustomPreStopHook(t1 *testing.T) {
	profile := buildBaseThinRuntimeProfile()
	profile.Spec.Fuse.Lifecycle = &corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"custom.sh"},
			},
		},
	}
	runtime := buildBaseThinRuntime()
	dataset := buildBaseDataset()
	wantValue := &ThinValue{
		Fuse: Fuse{
			Enabled:         true,
			Image:           "test",
			ImageTag:        "v1",
			ImagePullPolicy: "Always",
			TargetPath:      "/thin/fluid/test/thin-fuse",
			Resources: common.Resources{
				Requests: map[corev1.ResourceName]string{
					corev1.ResourceCPU:    "100m",
					corev1.ResourceMemory: "1Gi",
				},
				Limits: map[corev1.ResourceName]string{
					corev1.ResourceCPU:    "200m",
					corev1.ResourceMemory: "4Gi",
				},
			},
			HostNetwork: true,
			Envs: []corev1.EnvVar{{
				Name:  "a",
				Value: "b",
			}, {
				Name: "b",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-cm",
						},
					},
				},
			}, {
				Name:  common.ThinFuseOptionEnvKey,
				Value: "fuse-opt=foo",
			}, {
				Name:  common.ThinFusePointEnvKey,
				Value: "/thin/fluid/test/thin-fuse",
			}},
			NodeSelector: map[string]string{"b": "c", "fluid.io/f-fluid-test": "true"},
			Ports: []corev1.ContainerPort{{
				Name:          "port",
				ContainerPort: 8080,
			}},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
					},
				},
				InitialDelaySeconds: 1,
				TimeoutSeconds:      1,
				PeriodSeconds:       1,
				SuccessThreshold:    1,
				FailureThreshold:    1,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
					},
				},
				InitialDelaySeconds: 1,
				TimeoutSeconds:      1,
				PeriodSeconds:       1,
				SuccessThreshold:    1,
				FailureThreshold:    1,
			},
			Volumes: []corev1.Volume{{
				Name: "a",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/test"},
				},
			}, {
				Name: "b",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/b"},
				},
			}},
			VolumeMounts: []corev1.VolumeMount{{
				Name:      "a",
				MountPath: "/test",
			}, {
				Name:      "b",
				MountPath: "/b",
			}},
			// ConfigValue: "{\"/thin/fluid/test/thin-fuse\":\"a=b\"}",
			// MountPath:   "/thin/fluid/test/thin-fuse",
			ConfigValue:   "{\"mounts\":[{\"mountPoint\":\"abc\",\"options\":{\"a\":\"b\",\"c\":\"d\"}}],\"targetPath\":\"/thin/fluid/test/thin-fuse\",\"runtimeOptions\":{\"fuse-opt\":\"foo\"},\"accessModes\":[\"ReadWriteMany\"]}",
			ConfigStorage: "configmap",
			Lifecycle: &corev1.Lifecycle{
				PreStop: &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"custom.sh",
						},
					},
				},
			},
		},
	}
	value := &ThinValue{}
	runtimeInfo, err := base.BuildRuntimeInfo("test", "fluid", "thin")
	if err != nil {
		t1.Errorf("fail to create the runtimeInfo with error %v", err)
	}
	t1.Run("test", func(t1 *testing.T) {
		t := &ThinEngine{
			Log:         fake.NullLogger(),
			namespace:   "fluid",
			name:        "test",
			runtime:     runtime,
			runtimeInfo: runtimeInfo,
			Client:      fake.NewFakeClientWithScheme(testScheme),
		}
		if err := t.transformFuse(runtime, profile, dataset, value); err != nil {
			t1.Errorf("transformFuse() error = %v", err)
		}

		value.Fuse.Envs = testutil.SortEnvVarByName(value.Fuse.Envs, common.ThinFuseOptionEnvKey)
		if !testutil.DeepEqualIgnoringSliceOrder(t1, value.Fuse, wantValue.Fuse) {
			valueYaml, _ := yaml.Marshal(value.Fuse)
			wantYaml, _ := yaml.Marshal(wantValue.Fuse)
			t1.Errorf("transformFuse() \ngot = %v, \nwant = %v", string(valueYaml), string(wantYaml))
		}
	})
}

func TestThinEngine_transformFuseWithContainerNetwork(t1 *testing.T) {
	profile := buildBaseThinRuntimeProfile()
	thinRuntime := buildBaseThinRuntime()
	dataset := buildBaseDataset()
	value := &ThinValue{}
	runtimeInfo, err := base.BuildRuntimeInfo("test", "fluid", "thin")
	if err != nil {
		t1.Errorf("fail to create the runtimeInfo with error %v", err)
	}
	t1.Run("test", func(t1 *testing.T) {
		hostNetworkMode := datav1alpha1.HostNetworkMode
		containerNetworkMode := datav1alpha1.ContainerNetworkMode
		tests := []struct {
			name                     string
			fuseNetworkModeInProfile *datav1alpha1.NetworkMode
			fuseNetworkModeInRuntime *datav1alpha1.NetworkMode
			wantFuseHostNetwork      bool
		}{
			{
				name:                     "nilInProfile-nilInRuntime",
				fuseNetworkModeInProfile: nil,
				fuseNetworkModeInRuntime: nil,
				wantFuseHostNetwork:      true,
			}, {
				name:                     "containerNetworkInProfile-nilInRuntime",
				fuseNetworkModeInProfile: &containerNetworkMode,
				fuseNetworkModeInRuntime: nil,
				wantFuseHostNetwork:      false,
			},
			{
				name:                     "hostNetworkInProfile-nilInRuntime",
				fuseNetworkModeInProfile: &hostNetworkMode,
				fuseNetworkModeInRuntime: nil,
				wantFuseHostNetwork:      true,
			},
			{
				name:                     "nilInProfile-hostNetworkInRuntime",
				fuseNetworkModeInProfile: nil,
				fuseNetworkModeInRuntime: &hostNetworkMode,
				wantFuseHostNetwork:      true,
			},
			{
				name:                     "nilInProfile-containerNetworkInRuntime",
				fuseNetworkModeInProfile: nil,
				fuseNetworkModeInRuntime: &containerNetworkMode,
				wantFuseHostNetwork:      false,
			},
			{
				name:                     "containerNetworkInProfile-hostNetworkInRuntime",
				fuseNetworkModeInProfile: &containerNetworkMode,
				fuseNetworkModeInRuntime: &hostNetworkMode,
				wantFuseHostNetwork:      true,
			},
			{
				name:                     "hostNetworkInProfile-containerNetworkInRuntime",
				fuseNetworkModeInProfile: &hostNetworkMode,
				fuseNetworkModeInRuntime: &containerNetworkMode,
				wantFuseHostNetwork:      false,
			},
		}

		for _, test := range tests {
			if test.fuseNetworkModeInProfile != nil {
				profile.Spec.Fuse.NetworkMode = *test.fuseNetworkModeInProfile
			}
			if test.fuseNetworkModeInRuntime != nil {
				thinRuntime.Spec.Fuse.NetworkMode = *test.fuseNetworkModeInRuntime
			}
			t := &ThinEngine{
				Log:         fake.NullLogger(),
				namespace:   "fluid",
				name:        "test",
				runtime:     thinRuntime,
				runtimeInfo: runtimeInfo,
				Client:      fake.NewFakeClientWithScheme(testScheme),
			}
			if err := t.transformFuse(thinRuntime, profile, dataset, value); err != nil {
				t1.Errorf("transformFuse() error = %v", err)
			}

			if value.Fuse.HostNetwork != test.wantFuseHostNetwork {
				t1.Errorf("transformFuse() \ngot HostNetwork = %v, \nwant HostNetwork = %v", value.Fuse.HostNetwork, test.wantFuseHostNetwork)

			}
		}
	})
}

func TestThinEngine_transformFuseWithDuplicateOptionKey(t1 *testing.T) {
	profile := &datav1alpha1.ThinRuntimeProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: datav1alpha1.ThinRuntimeProfileSpec{
			FileSystemType: "test",
			Fuse: datav1alpha1.ThinFuseSpec{
				Image:           "test",
				ImageTag:        "v1",
				ImagePullPolicy: "Always",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						// Should be inherited
						corev1.ResourceCPU: resource.MustParse("100m"),
						// Should be overridden
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
				Env: []corev1.EnvVar{{
					Name:  "a",
					Value: "b",
				}},
				NodeSelector: map[string]string{"a": "b"},
				Ports: []corev1.ContainerPort{{
					Name:          "port",
					ContainerPort: 8080,
				}},
				NetworkMode: datav1alpha1.HostNetworkMode,
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "a",
					MountPath: "/test",
				}},
			},
			Volumes: []corev1.Volume{{
				Name: "a",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/test"},
				},
			}},
		},
	}
	runtime := &datav1alpha1.ThinRuntime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "fluid",
		},
		Spec: datav1alpha1.ThinRuntimeSpec{
			ThinRuntimeProfileName: "test",
			Fuse: datav1alpha1.ThinFuseSpec{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
				Env: []corev1.EnvVar{{
					Name: "b",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-cm"},
						},
					},
				}},
				NodeSelector: map[string]string{"b": "c"},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "b",
					MountPath: "/b",
				}},
				LivenessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/healthz",
						},
					},
					InitialDelaySeconds: 1,
					TimeoutSeconds:      1,
					PeriodSeconds:       1,
					SuccessThreshold:    1,
					FailureThreshold:    1,
				},
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/healthz",
						},
					},
					InitialDelaySeconds: 1,
					TimeoutSeconds:      1,
					PeriodSeconds:       1,
					SuccessThreshold:    1,
					FailureThreshold:    1,
				},
			},
			Volumes: []corev1.Volume{{
				Name: "b",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/b"},
				},
			}},
		},
	}
	dataset := &datav1alpha1.Dataset{
		Spec: datav1alpha1.DatasetSpec{
			SharedOptions: map[string]string{
				"a": "c",
			},
			Mounts: []datav1alpha1.Mount{{
				MountPoint: "abc",
				Options:    map[string]string{"a": "b"},
			}},
		},
	}
	wantValue := &ThinValue{
		Fuse: Fuse{
			Enabled:         true,
			Image:           "test",
			ImageTag:        "v1",
			ImagePullPolicy: "Always",
			TargetPath:      "/thin/fluid/test/thin-fuse",
			Resources: common.Resources{
				Requests: map[corev1.ResourceName]string{
					corev1.ResourceCPU:    "100m",
					corev1.ResourceMemory: "1Gi",
				},
				Limits: map[corev1.ResourceName]string{
					corev1.ResourceCPU:    "200m",
					corev1.ResourceMemory: "4Gi",
				},
			},
			HostNetwork: true,
			Envs: []corev1.EnvVar{{
				Name:  "a",
				Value: "b",
			}, {
				Name: "b",
				ValueFrom: &corev1.EnvVarSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-cm",
						},
					},
				},
			}, {
				Name:  common.ThinFusePointEnvKey,
				Value: "/thin/fluid/test/thin-fuse",
			}},
			NodeSelector: map[string]string{"b": "c", "fluid.io/f-fluid-test": "true"},
			Ports: []corev1.ContainerPort{{
				Name:          "port",
				ContainerPort: 8080,
			}},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
					},
				},
				InitialDelaySeconds: 1,
				TimeoutSeconds:      1,
				PeriodSeconds:       1,
				SuccessThreshold:    1,
				FailureThreshold:    1,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
					},
				},
				InitialDelaySeconds: 1,
				TimeoutSeconds:      1,
				PeriodSeconds:       1,
				SuccessThreshold:    1,
				FailureThreshold:    1,
			},
			Volumes: []corev1.Volume{{
				Name: "a",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/test"},
				},
			}, {
				Name: "b",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/b"},
				},
			}},
			VolumeMounts: []corev1.VolumeMount{{
				Name:      "a",
				MountPath: "/test",
			}, {
				Name:      "b",
				MountPath: "/b",
			}},
			// ConfigValue: "{\"/thin/fluid/test/thin-fuse\":\"a=b\"}",
			// MountPath:   "/thin/fluid/test/thin-fuse",
			ConfigValue:   "{\"mounts\":[{\"mountPoint\":\"abc\",\"options\":{\"a\":\"b\"}}],\"targetPath\":\"/thin/fluid/test/thin-fuse\",\"accessModes\":[\"ReadOnlyMany\"]}",
			ConfigStorage: "configmap",
			Lifecycle: &corev1.Lifecycle{
				PreStop: &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"umount",
							"/thin/fluid/test/thin-fuse",
						},
					},
				},
			},
		},
	}
	value := &ThinValue{}
	t1.Run("test", func(t1 *testing.T) {
		runtimeInfo, err := base.BuildRuntimeInfo("test", "fluid", "thin")
		if err != nil {
			t1.Errorf("fail to create the runtimeInfo with error %v", err)
		}

		t := &ThinEngine{
			Log:         fake.NullLogger(),
			namespace:   "fluid",
			name:        "test",
			runtime:     runtime,
			runtimeInfo: runtimeInfo,
			Client:      fake.NewFakeClientWithScheme(testScheme),
		}
		if err := t.transformFuse(runtime, profile, dataset, value); err != nil {
			t1.Errorf("transformFuse() error = %v", err)
		}

		value.Fuse.Envs = testutil.SortEnvVarByName(value.Fuse.Envs, common.ThinFuseOptionEnvKey)
		if !testutil.DeepEqualIgnoringSliceOrder(t1, value.Fuse, wantValue.Fuse) {
			valueYaml, _ := yaml.Marshal(value.Fuse)
			wantYaml, _ := yaml.Marshal(wantValue.Fuse)
			t1.Errorf("transformFuse() \ngot = %v, \nwant = %v", string(valueYaml), string(wantYaml))
		}
	})
}

func TestParseHostVolumeFromDataset(t *testing.T) {
	tests := []struct {
		name        string
		dataset     *datav1alpha1.Dataset
		expected    *ThinValue
		expectedErr bool
	}{
		{
			name: "ValidHostVolume",
			dataset: &datav1alpha1.Dataset{
				Spec: datav1alpha1.DatasetSpec{
					Mounts: []datav1alpha1.Mount{
						{
							Options: map[string]string{
								common.DatasetOptionFluidFuseHostVolume: "/host/path:/mount/path",
							},
						},
					},
				},
			},
			expected: &ThinValue{
				Fuse: Fuse{
					Volumes: []corev1.Volume{
						{
							Name: "fluid-fuse-hostvolume-0",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/host/path",
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "fluid-fuse-hostvolume-0",
							MountPath: "/mount/path",
						},
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "ValidHostVolumeWithDefaultMountPath",
			dataset: &datav1alpha1.Dataset{
				Spec: datav1alpha1.DatasetSpec{
					Mounts: []datav1alpha1.Mount{
						{
							Options: map[string]string{
								common.DatasetOptionFluidFuseHostVolume: "/host/path",
							},
						},
					},
				},
			},
			expected: &ThinValue{
				Fuse: Fuse{
					Volumes: []corev1.Volume{
						{
							Name: "fluid-fuse-hostvolume-0",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/host/path",
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "fluid-fuse-hostvolume-0",
							MountPath: "/host/path",
						},
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "ValidHostVolumeWithMultiHostVolume",
			dataset: &datav1alpha1.Dataset{
				Spec: datav1alpha1.DatasetSpec{
					Mounts: []datav1alpha1.Mount{
						{
							Options: map[string]string{
								common.DatasetOptionFluidFuseHostVolume: "/host/path-1",
							},
						},
						{
							Options: map[string]string{
								common.DatasetOptionFluidFuseHostVolume: "/host/path-2",
							},
						},
					},
				},
			},
			expected: &ThinValue{
				Fuse: Fuse{
					Volumes: []corev1.Volume{
						{
							Name: "fluid-fuse-hostvolume-0",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/host/path-1",
								},
							},
						},
						{
							Name: "fluid-fuse-hostvolume-1",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/host/path-2",
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "fluid-fuse-hostvolume-0",
							MountPath: "/host/path-1",
						},
						{
							Name:      "fluid-fuse-hostvolume-1",
							MountPath: "/host/path-2",
						},
					},
				},
			},
			expectedErr: false,
		},
		{
			name: "InvalidHostVolume",
			dataset: &datav1alpha1.Dataset{
				Spec: datav1alpha1.DatasetSpec{
					Mounts: []datav1alpha1.Mount{
						{
							Options: map[string]string{
								common.DatasetOptionFluidFuseHostVolume: "/host/path:/mount/path:/invalid",
							},
						},
					},
				},
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name: "NonAbsoluteHostPath",
			dataset: &datav1alpha1.Dataset{
				Spec: datav1alpha1.DatasetSpec{
					Mounts: []datav1alpha1.Mount{
						{
							Options: map[string]string{
								common.DatasetOptionFluidFuseHostVolume: "host/path:/mount/path",
							},
						},
					},
				},
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name: "NonAbsoluteMountPath",
			dataset: &datav1alpha1.Dataset{
				Spec: datav1alpha1.DatasetSpec{
					Mounts: []datav1alpha1.Mount{
						{
							Options: map[string]string{
								common.DatasetOptionFluidFuseHostVolume: "/host/path:mount/path",
							},
						},
					},
				},
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name: "ReadOnlyAccessMode",
			dataset: &datav1alpha1.Dataset{
				Spec: datav1alpha1.DatasetSpec{
					Mounts: []datav1alpha1.Mount{
						{
							Options: map[string]string{
								common.DatasetOptionFluidFuseHostVolume: "/host/path:/mount/path",
							},
						},
					},
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadOnlyMany,
					},
				},
			},
			expected: &ThinValue{
				Fuse: Fuse{
					Volumes: []corev1.Volume{
						{
							Name: "fluid-fuse-hostvolume-0",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/host/path",
								},
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "fluid-fuse-hostvolume-0",
							MountPath: "/mount/path",
							ReadOnly:  true,
						},
					},
				},
			},
			expectedErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			thinValue := &ThinValue{}

			thinEngine := &ThinEngine{
				Log:       fake.NullLogger(),
				namespace: "fluid",
				name:      "test",
				Client:    fake.NewFakeClientWithScheme(testScheme),
			}

			err := thinEngine.parseHostVolumeFromDataset(test.dataset, thinValue)
			if err != nil != test.expectedErr {
				t.Errorf("Expected error %v, got %v", test.expectedErr, err)
			}
			if err == nil && !reflect.DeepEqual(thinValue, test.expected) {
				t.Errorf("Expected %v, got %v", test.expected, thinValue)
			}
		})
	}
}

func buildBaseDataset() *datav1alpha1.Dataset {
	return &datav1alpha1.Dataset{
		Spec: datav1alpha1.DatasetSpec{
			SharedOptions: map[string]string{
				"c": "d",
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
			Mounts: []datav1alpha1.Mount{{
				MountPoint: "abc",
				Options:    map[string]string{"a": "b"},
			}},
		},
	}
}

func buildBaseThinRuntime() *datav1alpha1.ThinRuntime {
	return &datav1alpha1.ThinRuntime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "fluid",
		},
		Spec: datav1alpha1.ThinRuntimeSpec{
			ThinRuntimeProfileName: "test",
			Fuse: datav1alpha1.ThinFuseSpec{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
				Env: []corev1.EnvVar{{
					Name: "b",
					ValueFrom: &corev1.EnvVarSource{
						ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-cm"},
						},
					},
				}},
				Options: map[string]string{
					"fuse-opt": "foo",
				},
				NodeSelector: map[string]string{"b": "c"},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "b",
					MountPath: "/b",
				}},
				LivenessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/healthz",
						},
					},
					InitialDelaySeconds: 1,
					TimeoutSeconds:      1,
					PeriodSeconds:       1,
					SuccessThreshold:    1,
					FailureThreshold:    1,
				},
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/healthz",
						},
					},
					InitialDelaySeconds: 1,
					TimeoutSeconds:      1,
					PeriodSeconds:       1,
					SuccessThreshold:    1,
					FailureThreshold:    1,
				},
			},
			Volumes: []corev1.Volume{{
				Name: "b",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/b"},
				},
			}},
		},
	}
}

func buildBaseThinRuntimeProfile() *datav1alpha1.ThinRuntimeProfile {
	return &datav1alpha1.ThinRuntimeProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: datav1alpha1.ThinRuntimeProfileSpec{
			FileSystemType: "test",
			Fuse: datav1alpha1.ThinFuseSpec{
				Image:           "test",
				ImageTag:        "v1",
				ImagePullPolicy: "Always",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						// Should be inherited
						corev1.ResourceCPU: resource.MustParse("100m"),
						// Should be overridden
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
				},
				Env: []corev1.EnvVar{{
					Name:  "a",
					Value: "b",
				}},
				NodeSelector: map[string]string{"a": "b"},
				Ports: []corev1.ContainerPort{{
					Name:          "port",
					ContainerPort: 8080,
				}},
				NetworkMode: datav1alpha1.HostNetworkMode,
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "a",
					MountPath: "/test",
				}},
			},
			Volumes: []corev1.Volume{{
				Name: "a",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{Path: "/test"},
				},
			}},
		},
	}
}
