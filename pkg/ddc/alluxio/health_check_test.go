/*

Copyright 2021 The Fluid Author.

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

package alluxio

import (
	"context"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"

	datav1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	"github.com/fluid-cloudnative/fluid/pkg/common"
	"github.com/fluid-cloudnative/fluid/pkg/ctrl"
	"github.com/fluid-cloudnative/fluid/pkg/ddc/base"
	"github.com/fluid-cloudnative/fluid/pkg/utils"
	"github.com/fluid-cloudnative/fluid/pkg/utils/fake"
)

// TestCheckRuntimeHealthy is a unit test that verifies the health check logic
// for the Jindo runtime environment. It creates a mock runtime and simulates
// different health scenarios, asserting that the status returned by the
// CheckRuntimeHealthy method matches the expected outcome.
func TestCheckRuntimeHealthy(t *testing.T) {
	var statefulsetInputs = []appsv1.StatefulSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase-master",
				Namespace: "fluid",
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:      3,
				ReadyReplicas: 3,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase-worker",
				Namespace: "fluid",
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:        1,
				ReadyReplicas:   1,
				CurrentReplicas: 1,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: ptr.To[int32](1),
			},
		},
	}
	testObjs := []runtime.Object{}
	for _, statefulset := range statefulsetInputs {
		testObjs = append(testObjs, statefulset.DeepCopy())
	}

	var daemonSetInputs = []appsv1.DaemonSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase-fuse",
				Namespace: "fluid",
			},
			Status: appsv1.DaemonSetStatus{
				NumberUnavailable: 0,
				NumberReady:       1,
				NumberAvailable:   1,
			},
		},
	}
	for _, daemonSet := range daemonSetInputs {
		testObjs = append(testObjs, daemonSet.DeepCopy())
	}

	var alluxioruntimeInputs = []datav1alpha1.AlluxioRuntime{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase",
				Namespace: "fluid",
			},
			Spec: datav1alpha1.AlluxioRuntimeSpec{
				Master: datav1alpha1.AlluxioCompTemplateSpec{
					Replicas: 1,
				},
			},
			Status: datav1alpha1.RuntimeStatus{
				CacheStates: map[common.CacheStateName]string{
					common.Cached: "true",
				},
			},
		},
	}
	for _, alluxioruntime := range alluxioruntimeInputs {
		testObjs = append(testObjs, alluxioruntime.DeepCopy())
	}

	var datasetInputs = []*datav1alpha1.Dataset{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase",
				Namespace: "fluid",
			},
			Spec: datav1alpha1.DatasetSpec{},
			Status: datav1alpha1.DatasetStatus{
				HCFSStatus: &datav1alpha1.HCFSStatus{
					Endpoint:                    "test Endpoint",
					UnderlayerFileSystemVersion: "Underlayer HCFS Compatible Version",
				},
			},
		},
	}
	for _, dataset := range datasetInputs {
		testObjs = append(testObjs, dataset.DeepCopy())
	}

	client := fake.NewFakeClientWithScheme(testScheme, testObjs...)

	engines := []AlluxioEngine{
		{
			Client:    client,
			Log:       fake.NullLogger(),
			namespace: "fluid",
			name:      "hbase",
			runtime:   &alluxioruntimeInputs[0],
			Recorder:  record.NewFakeRecorder(1),
		},
	}

	var testCase = []struct {
		engine                             AlluxioEngine
		expectedErrorNil                   bool
		expectedMasterPhase                datav1alpha1.RuntimePhase
		expectedWorkerPhase                datav1alpha1.RuntimePhase
		expectedRuntimeWorkerNumberReady   int32
		expectedRuntimeWorkerAvailable     int32
		expectedRuntimeFuseNumberReady     int32
		expectedRuntimeFuseNumberAvailable int32
		expectedDataset                    datav1alpha1.Dataset
	}{
		{
			engine:                             engines[0],
			expectedErrorNil:                   true,
			expectedMasterPhase:                datav1alpha1.RuntimePhaseReady,
			expectedWorkerPhase:                "",
			expectedRuntimeWorkerNumberReady:   1,
			expectedRuntimeWorkerAvailable:     1,
			expectedRuntimeFuseNumberReady:     1,
			expectedRuntimeFuseNumberAvailable: 1,
			expectedDataset: datav1alpha1.Dataset{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hbase",
					Namespace: "fluid",
				},
				Status: datav1alpha1.DatasetStatus{
					Phase: datav1alpha1.BoundDatasetPhase,
					CacheStates: map[common.CacheStateName]string{
						common.Cached: "true",
					},
					HCFSStatus: &datav1alpha1.HCFSStatus{
						Endpoint:                    "test Endpoint",
						UnderlayerFileSystemVersion: "Underlayer HCFS Compatible Version",
					},
				},
			},
		},
	}
	for _, test := range testCase {
		runtimeInfo, _ := base.BuildRuntimeInfo(test.engine.name, test.engine.namespace, common.AlluxioRuntime)
		test.engine.Helper = ctrl.BuildHelper(runtimeInfo, client, test.engine.Log)
		err := test.engine.CheckRuntimeHealthy()
		if err != nil && test.expectedErrorNil == true ||
			err == nil && test.expectedErrorNil == false {
			t.Errorf("fail to exec the checkMasterHealthy function with err %v", err)
			return
		}
		if test.expectedErrorNil == false {
			continue
		}

		alluxioruntime, err := test.engine.getRuntime()
		if err != nil {
			t.Errorf("fail to get the runtime with the error %v", err)
			return
		}
		if alluxioruntime.Status.MasterPhase != test.expectedMasterPhase {
			t.Errorf("fail to update the runtime status, get %s, expect %s", alluxioruntime.Status.MasterPhase, test.expectedMasterPhase)
			return
		}
		if alluxioruntime.Status.WorkerNumberReady != test.expectedRuntimeWorkerNumberReady ||
			alluxioruntime.Status.WorkerNumberAvailable != test.expectedRuntimeWorkerAvailable {
			t.Errorf("fail to update the runtime")
			return
		}
		if alluxioruntime.Status.FuseNumberReady != test.expectedRuntimeFuseNumberReady ||
			alluxioruntime.Status.FuseNumberAvailable != test.expectedRuntimeFuseNumberAvailable {
			t.Errorf("fail to update the runtime")
			return
		}

		_, cond := utils.GetRuntimeCondition(alluxioruntime.Status.Conditions, datav1alpha1.RuntimeMasterReady)
		if cond == nil {
			t.Errorf("fail to update the condition")
			return
		}
		_, cond = utils.GetRuntimeCondition(alluxioruntime.Status.Conditions, datav1alpha1.RuntimeWorkersReady)
		if cond == nil {
			t.Errorf("fail to update the condition")
			return
		}
		_, cond = utils.GetRuntimeCondition(alluxioruntime.Status.Conditions, datav1alpha1.RuntimeFusesReady)
		if cond == nil {
			t.Errorf("fail to update the condition")
			return
		}

		var datasets datav1alpha1.DatasetList
		err = client.List(context.TODO(), &datasets)
		if err != nil {
			t.Errorf("fail to list the datasets with error %v", err)
			return
		}
		if !reflect.DeepEqual(datasets.Items[0].Status.Phase, test.expectedDataset.Status.Phase) ||
			!reflect.DeepEqual(datasets.Items[0].Status.CacheStates, test.expectedDataset.Status.CacheStates) ||
			!reflect.DeepEqual(datasets.Items[0].Status.HCFSStatus, test.expectedDataset.Status.HCFSStatus) {
			t.Errorf("fail to exec the function with error %v", err)
			return
		}
	}
}

// TestCheckMasterHealthy is a test function that verifies the health status of master nodes
// in a simulated Kubernetes environment. It uses fake clients to create test StatefulSets
// and AlluxioRuntime objects, and checks if the master nodes are healthy based on predefined
// conditions. The test cases include scenarios for different engines (HBase and Spark) with
// expected outcomes for error presence and master phase status.
func TestCheckMasterHealthy(t *testing.T) {
	var statefulsetInputs = []appsv1.StatefulSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase-master",
				Namespace: "fluid",
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:      3,
				ReadyReplicas: 2,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spark-master",
				Namespace: "fluid",
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:      3,
				ReadyReplicas: 3,
			},
		},
	}

	testObjs := []runtime.Object{}
	for _, statefulset := range statefulsetInputs {
		testObjs = append(testObjs, statefulset.DeepCopy())
	}

	var alluxioruntimeInputs = []datav1alpha1.AlluxioRuntime{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase",
				Namespace: "fluid",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spark",
				Namespace: "fluid",
			},
		},
	}
	for _, alluxioruntimeInput := range alluxioruntimeInputs {
		testObjs = append(testObjs, alluxioruntimeInput.DeepCopy())
	}
	client := fake.NewFakeClientWithScheme(testScheme, testObjs...)

	engines := []AlluxioEngine{
		{
			Client:    client,
			Log:       fake.NullLogger(),
			namespace: "fluid",
			name:      "hbase",
			runtime: &datav1alpha1.AlluxioRuntime{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hbase",
					Namespace: "fluid",
				},
			},
		},
		{
			Client:    client,
			Log:       fake.NullLogger(),
			namespace: "fluid",
			name:      "spark",
			runtime: &datav1alpha1.AlluxioRuntime{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "spark",
					Namespace: "fluid",
				},
			},
		},
	}

	var testCase = []struct {
		engine              AlluxioEngine
		expectedErrorNil    bool
		expectedMasterPhase datav1alpha1.RuntimePhase
	}{
		{
			engine:              engines[0],
			expectedErrorNil:    false,
			expectedMasterPhase: "",
		},
		{
			engine:              engines[1],
			expectedErrorNil:    true,
			expectedMasterPhase: datav1alpha1.RuntimePhaseReady,
		},
	}

	for _, test := range testCase {
		err := test.engine.checkMasterHealthy()
		if err != nil && test.expectedErrorNil == true ||
			err == nil && test.expectedErrorNil == false {
			t.Errorf("fail to exec the checkMasterHealthy function with err %v", err)
			return
		}

		if test.expectedErrorNil == false {
			continue
		}
		alluxioruntime, err := test.engine.getRuntime()
		if err != nil {
			t.Errorf("fail to get the runtime with the error %v", err)
			return
		}

		if alluxioruntime.Status.MasterPhase != test.expectedMasterPhase {
			t.Errorf("fail to update the runtime status, get %s, expect %s", alluxioruntime.Status.MasterPhase, test.expectedMasterPhase)
			return
		}

		_, cond := utils.GetRuntimeCondition(alluxioruntime.Status.Conditions, datav1alpha1.RuntimeMasterReady)
		if cond == nil {
			t.Errorf("fail to update the condition")
			return
		}
	}
}

// TestCheckWorkersHealthy validates the health check functionality for Alluxio runtime workers.
// It covers three primary scenarios:
// 1. Unhealthy workers (StatefulSet with readyReplicas < desired replicas)
// 2. Healthy workers (StatefulSet with readyReplicas matching desired count)
// 3. Deprecated worker types (DaemonSet-based workers that should remain unchanged)
//
// The test verifies:
// - Correct error handling based on worker status
// - Proper updates to RuntimeStatus fields (WorkerNumberReady/Available)
// - Appropriate condition updates for worker readiness
// - Idempotency for deprecated worker implementations
//
// Test cases include:
//   - Case 1: Workers not ready (0 ready replicas out of 2 desired)
//   - Case 2: Workers fully ready (1 ready replica matching desired count)
//   - Case 3: Deprecated worker type (no status changes expected)
func TestCheckWorkersHealthy(t *testing.T) {
	var statefulSetInputs = []appsv1.StatefulSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase-worker",
				Namespace: "fluid",
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:        1,
				ReadyReplicas:   0,
				CurrentReplicas: 1,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: ptr.To[int32](2),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spark-worker",
				Namespace: "fluid",
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:        1,
				ReadyReplicas:   1,
				CurrentReplicas: 1,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: ptr.To[int32](1),
			},
		},
	}

	var daemonSetInputs = []appsv1.DaemonSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deprecated-worker",
				Namespace: "fluid",
			},
		},
	}

	testObjs := []runtime.Object{}
	for _, statefulSet := range statefulSetInputs {
		testObjs = append(testObjs, statefulSet.DeepCopy())
	}

	for _, daemonSet := range daemonSetInputs {
		testObjs = append(testObjs, daemonSet.DeepCopy())
	}

	var alluxioruntimeInputs = []datav1alpha1.AlluxioRuntime{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase",
				Namespace: "fluid",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spark",
				Namespace: "fluid",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deprecated",
				Namespace: "fluid",
			},
		},
	}
	for _, alluxioruntimeInput := range alluxioruntimeInputs {
		testObjs = append(testObjs, alluxioruntimeInput.DeepCopy())
	}
	client := fake.NewFakeClientWithScheme(testScheme, testObjs...)

	engines := []AlluxioEngine{
		{
			Client:    client,
			Log:       fake.NullLogger(),
			namespace: "fluid",
			name:      "hbase",
			runtime: &datav1alpha1.AlluxioRuntime{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hbase",
					Namespace: "fluid",
				},
			},
		},
		{
			Client:    client,
			Log:       fake.NullLogger(),
			namespace: "fluid",
			name:      "spark",
			runtime: &datav1alpha1.AlluxioRuntime{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "spark",
					Namespace: "fluid",
				},
			},
		},
		{
			Client:    client,
			Log:       fake.NullLogger(),
			namespace: "fluid",
			name:      "deprecated",
			runtime: &datav1alpha1.AlluxioRuntime{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deprecated",
					Namespace: "fluid",
				},
			},
			Recorder: record.NewFakeRecorder(1),
		},
	}

	var testCase = []struct {
		engine                           AlluxioEngine
		expectedWorkerPhase              datav1alpha1.RuntimePhase
		expectedErrorNil                 bool
		expectedRuntimeWorkerNumberReady int32
		expectedRuntimeWorkerAvailable   int32
	}{
		{
			engine:                           engines[0],
			expectedWorkerPhase:              datav1alpha1.RuntimePhaseNotReady,
			expectedErrorNil:                 false,
			expectedRuntimeWorkerNumberReady: 0,
			expectedRuntimeWorkerAvailable:   1,
		},
		{
			engine:                           engines[1],
			expectedWorkerPhase:              "",
			expectedErrorNil:                 true,
			expectedRuntimeWorkerNumberReady: 1,
			expectedRuntimeWorkerAvailable:   1,
		},
	}

	for _, test := range testCase {
		err := test.engine.checkWorkersHealthy()
		if err != nil && test.expectedErrorNil == true ||
			err == nil && test.expectedErrorNil == false {
			t.Errorf("fail to exec the checkMasterHealthy function with err %v", err)
			return
		}

		alluxioruntime, err := test.engine.getRuntime()
		if err != nil {
			t.Errorf("fail to get the runtime with the error %v", err)
			return
		}

		if alluxioruntime.Status.WorkerNumberReady != test.expectedRuntimeWorkerNumberReady ||
			alluxioruntime.Status.WorkerNumberAvailable != test.expectedRuntimeWorkerAvailable {
			t.Errorf("fail to update the runtime")
			return
		}

		_, cond := utils.GetRuntimeCondition(alluxioruntime.Status.Conditions, datav1alpha1.RuntimeWorkersReady)
		if cond == nil {
			t.Errorf("fail to update the condition")
			return
		}
	}

	var testCaseWithDeprecatedRuntime = []struct {
		engine           AlluxioEngine
		expectedErrorNil bool
	}{
		{
			engine:           engines[2],
			expectedErrorNil: true,
		},
	}

	for _, test := range testCaseWithDeprecatedRuntime {
		alluxioruntimeBefore, err := test.engine.getRuntime()
		if err != nil {
			t.Errorf("fail to get the runtime with the error %v", err)
			return
		}

		err = test.engine.checkWorkersHealthy()
		if test.expectedErrorNil != (err == nil) {
			t.Errorf("fail to exec the checkMasterHealthy function with err %v", err)
			return
		}

		alluxioruntimeAfter, err := test.engine.getRuntime()
		if err != nil {
			t.Errorf("fail to get the runtime with the error %v", err)
			return
		}

		if !reflect.DeepEqual(alluxioruntimeBefore, alluxioruntimeAfter) {
			t.Error("Runtime should remain unchanged")
			return
		}
	}
}

func TestCheckFuseHealthy(t *testing.T) {
	var daemonSetInputs = []appsv1.DaemonSet{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase-fuse",
				Namespace: "fluid",
			},
			Status: appsv1.DaemonSetStatus{
				NumberUnavailable: 1,
				NumberReady:       1,
				NumberAvailable:   1,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spark-fuse",
				Namespace: "fluid",
			},
			Status: appsv1.DaemonSetStatus{
				NumberUnavailable: 0,
				NumberReady:       1,
				NumberAvailable:   1,
			},
		},
	}

	testObjs := []runtime.Object{}
	for _, daemonSet := range daemonSetInputs {
		testObjs = append(testObjs, daemonSet.DeepCopy())
	}

	var alluxioruntimeInputs = []datav1alpha1.AlluxioRuntime{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hbase",
				Namespace: "fluid",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "spark",
				Namespace: "fluid",
			},
		},
	}
	for _, alluxioruntimeInput := range alluxioruntimeInputs {
		testObjs = append(testObjs, alluxioruntimeInput.DeepCopy())
	}
	client := fake.NewFakeClientWithScheme(testScheme, testObjs...)

	engines := []AlluxioEngine{
		{
			Client:    client,
			Log:       fake.NullLogger(),
			namespace: "fluid",
			name:      "hbase",
			runtime: &datav1alpha1.AlluxioRuntime{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "hbase",
					Namespace: "fluid",
				},
			},
			Recorder: record.NewFakeRecorder(1),
		},
		{
			Client:    client,
			Log:       fake.NullLogger(),
			namespace: "fluid",
			name:      "spark",
			runtime: &datav1alpha1.AlluxioRuntime{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "spark",
					Namespace: "fluid",
				},
			},
			Recorder: record.NewFakeRecorder(1),
		},
	}

	var testCase = []struct {
		engine                               AlluxioEngine
		expectedWorkerPhase                  datav1alpha1.RuntimePhase
		expectedErrorNil                     bool
		expectedRuntimeFuseNumberReady       int32
		expectedRuntimeFuseNumberAvailable   int32
		expectedRuntimeFuseNumberUnavailable int32
	}{
		{
			engine:                               engines[0],
			expectedWorkerPhase:                  datav1alpha1.RuntimePhaseNotReady,
			expectedErrorNil:                     true,
			expectedRuntimeFuseNumberReady:       1,
			expectedRuntimeFuseNumberAvailable:   1,
			expectedRuntimeFuseNumberUnavailable: 1,
		},
		{
			engine:                               engines[1],
			expectedWorkerPhase:                  "",
			expectedErrorNil:                     true,
			expectedRuntimeFuseNumberReady:       1,
			expectedRuntimeFuseNumberAvailable:   1,
			expectedRuntimeFuseNumberUnavailable: 0,
		},
	}

	for _, test := range testCase {
		runtimeInfo, _ := base.BuildRuntimeInfo(test.engine.name, test.engine.namespace, common.AlluxioRuntime)
		test.engine.Helper = ctrl.BuildHelper(runtimeInfo, client, test.engine.Log)
		err := test.engine.checkFuseHealthy()
		if err != nil && test.expectedErrorNil == true ||
			err == nil && test.expectedErrorNil == false {
			t.Errorf("fail to exec the checkMasterHealthy function with err %v", err)
			return
		}

		alluxioruntime, err := test.engine.getRuntime()
		if err != nil {
			t.Errorf("fail to get the runtime with the error %v", err)
			return
		}

		if alluxioruntime.Status.FuseNumberReady != test.expectedRuntimeFuseNumberReady ||
			alluxioruntime.Status.FuseNumberAvailable != test.expectedRuntimeFuseNumberAvailable ||
			alluxioruntime.Status.FuseNumberUnavailable != test.expectedRuntimeFuseNumberUnavailable {
			t.Errorf("fail to update the runtime")
			return
		}

		_, cond := utils.GetRuntimeCondition(alluxioruntime.Status.Conditions, datav1alpha1.RuntimeFusesReady)
		if cond == nil {
			t.Errorf("fail to update the condition")
			return
		}
	}
}
