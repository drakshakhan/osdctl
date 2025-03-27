package hive

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	hivev1 "github.com/openshift/hive/apis/hive/v1"
	"github.com/openshift/hive/apis/hiveinternal/v1alpha1"
	mockk8s "github.com/openshift/osdctl/cmd/hive/clusterdeployment/mock/k8s"
)

func TestPrintFailingCluster(t *testing.T) {
	tests := []struct {
		name        string
		cdList      *hivev1.ClusterDeploymentList
		csList      *v1alpha1.ClusterSyncList
		expectError bool
	}{
		{
			name: "Successful_Execution",
			cdList: &hivev1.ClusterDeploymentList{
				Items: []hivev1.ClusterDeployment{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster",
						Namespace: "uhc-production-test",
					},
				}},
			},
			csList: &v1alpha1.ClusterSyncList{
				Items: []v1alpha1.ClusterSync{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-sync",
						Namespace: "uhc-production-test",
					},
				}},
			},
			expectError: false,
		},
		{
			name: "Missing_ClusterDeployment",
			cdList: &hivev1.ClusterDeploymentList{
				Items: []hivev1.ClusterDeployment{},
			},
			csList: &v1alpha1.ClusterSyncList{
				Items: []v1alpha1.ClusterSync{},
			},
			expectError: true,
		},
		{
			name: "SyncSet_Failure",
			cdList: &hivev1.ClusterDeploymentList{
				Items: []hivev1.ClusterDeployment{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster",
						Namespace: "uhc-production-test",
					},
				}},
			},
			csList: &v1alpha1.ClusterSyncList{
				Items: []v1alpha1.ClusterSync{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-cluster-sync",
						Namespace: "uhc-production-test",
					},
					Status: v1alpha1.ClusterSyncStatus{
						SyncSets: []v1alpha1.SyncStatus{{
							Name:           "sync-failure",
							Result:         "Failure",
							FailureMessage: "Some error occurred",
						}},
					},
				}},
			},
			expectError: false, //No error is returned, even though there is a failure, because the "failure" is handled by printing the error message instead of returning it as an error.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			if err := hivev1.AddToScheme(scheme); err != nil {
				t.Fatalf("Failed to add ClusterDeployment to scheme: %v", err)
			}

			if err := v1alpha1.AddToScheme(scheme); err != nil {
				t.Fatalf("Failed to add ClusterSync to scheme: %v", err)
			}

			// Add ClusterDeployments and ClusterSyncs to the fake client
			objects := []client.Object{}
			for _, cd := range tt.cdList.Items {
				objects = append(objects, &cd)
			}
			for _, cs := range tt.csList.Items {
				objects = append(objects, &cs)
			}

			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				Build()

			o := &clusterSyncFailuresOptions{
				kubeCli:   client,
				clusterID: "test",
			}

			err := o.printFailingCluster()
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

		})
	}
}

func TestListFailingClusterSyncs(t *testing.T) {
	testCases := []struct {
		name           string
		errorToReturn  error
		expectedResult []failingClusterSync
		isEmpty        bool
	}{
		{
			name:          "Success scenario with expected results",
			errorToReturn: nil,
			expectedResult: []failingClusterSync{
				{
					Name:            "example-clustersync",
					Namespace:       "uhc-production-1234",
					Timestamp:       time.Now().Format(time.RFC3339),
					LimitedSupport:  false,
					Hibernating:     true,
					FailingSyncSets: "selectorsyncset1 syncset1 ",
					ErrorMessage:    "Failed to sync selectorsyncset1\n\nFailed to sync syncset1\n\n",
				},
			},
			isEmpty: false,
		},
		{
			name:           "Empty_results_scenario(List_returns_no_items)",
			errorToReturn:  nil,
			expectedResult: []failingClusterSync{}, // Expecting empty results
			isEmpty:        true,
		},
		{
			name:           "Error_scenario",
			errorToReturn:  errors.New("failed to list ClusterSync resources due to network timeout"), // Triggering error condition
			expectedResult: nil,
			isEmpty:        false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockClient := mockk8s.NewMockClient(mockCtrl)

			// Setup mock data and mock client behavior directly
			cdList := hivev1.ClusterDeploymentList{
				Items: []hivev1.ClusterDeployment{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "example-cluster",
							Namespace: "uhc-production-1234",
						},
						Status: hivev1.ClusterDeploymentStatus{
							Conditions: []hivev1.ClusterDeploymentCondition{
								{
									Type:   "Hibernating",
									Status: corev1.ConditionTrue,
								},
							},
						},
					},
				},
			}

			csList := v1alpha1.ClusterSyncList{
				Items: []v1alpha1.ClusterSync{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "example-clustersync",
							Namespace: "uhc-production-1234",
						},
						Status: v1alpha1.ClusterSyncStatus{
							Conditions: []v1alpha1.ClusterSyncCondition{
								{
									Type:               "Ready",
									Status:             corev1.ConditionFalse,
									Reason:             "Failure",
									LastTransitionTime: metav1.Time{Time: time.Now()},
								},
							},
							SyncSets: []v1alpha1.SyncStatus{
								{
									Name:           "syncset1",
									Result:         "Failure",
									FailureMessage: "Failed to sync syncset1",
								},
							},
							SelectorSyncSets: []v1alpha1.SyncStatus{
								{
									Name:           "selectorsyncset1",
									Result:         "Failure",
									FailureMessage: "Failed to sync selectorsyncset1",
								},
							},
						},
					},
				},
			}

			if tc.isEmpty {
				csList = v1alpha1.ClusterSyncList{} // Return empty ClusterSyncList for the empty case
			}

			callTimes := 1
			if tc.errorToReturn == nil {
				callTimes = 2 // We expect two calls only if there's no error
			}

			// Setting up mock client expectations directly
			mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
				func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
					switch v := list.(type) {
					case *hivev1.ClusterDeploymentList:
						*v = cdList
					case *v1alpha1.ClusterSyncList:
						*v = csList
					}
					return tc.errorToReturn
				}).Times(callTimes) // Expect List to be called n times based on the error condition

			
			options := &clusterSyncFailuresOptions{
				kubeCli: mockClient,
			}

			result, err := options.listFailingClusterSyncs()

			if tc.errorToReturn != nil {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				if tc.isEmpty {
					// If isEmpty is true, the result should be an empty slice
					assert.Len(t, result, 0)
				} else {
					assert.NotNil(t, result)
					assert.Len(t, result, 1) // Since we expect only one result
					assert.Contains(t, result[0].ErrorMessage, "Failed to sync syncset1")
					assert.Contains(t, result[0].ErrorMessage, "Failed to sync selectorsyncset1")

					assert.Equal(t, "selectorsyncset1 syncset1 ", result[0].FailingSyncSets)

					assert.True(t, result[0].Hibernating)
					assert.False(t, result[0].LimitedSupport)
				}
			}
		})
	}
}

func TestSortBy(t *testing.T) {

	tests := []struct {
		name      string
		sortField string
		sortOrder string
		expected  []failingClusterSync
	}{
		// Test sorting by name in ascending order
		{
			name:      "sort_by_name_ascending_order",
			sortField: "name",
			sortOrder: "asc",
			expected: []failingClusterSync{
				{Name: "alpha", Timestamp: "2022-01-01T00:00:00Z", FailingSyncSets: "syncset1"},
				{Name: "beta", Timestamp: "2023-02-01T00:00:00Z", FailingSyncSets: "syncset2"},
				{Name: "zeta", Timestamp: "2023-01-01T00:00:00Z", FailingSyncSets: "syncset3"},
			},
		},
		// Test sorting by name in descending order
		{
			name:      "sort_by_name_descending_order",
			sortField: "name",
			sortOrder: "desc",
			expected: []failingClusterSync{
				{Name: "zeta", Timestamp: "2023-01-01T00:00:00Z", FailingSyncSets: "syncset3"},
				{Name: "beta", Timestamp: "2023-02-01T00:00:00Z", FailingSyncSets: "syncset2"},
				{Name: "alpha", Timestamp: "2022-01-01T00:00:00Z", FailingSyncSets: "syncset1"},
			},
		},
		// Test sorting by timestamp in ascending order
		{
			name:      "sort_by_timestamp_ascending_order",
			sortField: "timestamp",
			sortOrder: "asc",
			expected: []failingClusterSync{
				{Name: "alpha", Timestamp: "2022-01-01T00:00:00Z", FailingSyncSets: "syncset1"},
				{Name: "zeta", Timestamp: "2023-01-01T00:00:00Z", FailingSyncSets: "syncset3"},
				{Name: "beta", Timestamp: "2023-02-01T00:00:00Z", FailingSyncSets: "syncset2"},
			},
		},
		// Test sorting by timestamp in descending order
		{
			name:      "sort_by_timestamp_descending_order",
			sortField: "timestamp",
			sortOrder: "desc",
			expected: []failingClusterSync{
				{Name: "beta", Timestamp: "2023-02-01T00:00:00Z", FailingSyncSets: "syncset2"},
				{Name: "zeta", Timestamp: "2023-01-01T00:00:00Z", FailingSyncSets: "syncset3"},
				{Name: "alpha", Timestamp: "2022-01-01T00:00:00Z", FailingSyncSets: "syncset1"},
			},
		},
		// Test sorting by failingSyncSets in ascending order
		{
			name:      "sort_by_failingSyncSets_ascending_order",
			sortField: "failingsyncsets",
			sortOrder: "asc",
			expected: []failingClusterSync{
				{Name: "alpha", Timestamp: "2022-01-01T00:00:00Z", FailingSyncSets: "syncset1"},
				{Name: "beta", Timestamp: "2023-02-01T00:00:00Z", FailingSyncSets: "syncset2"},
				{Name: "zeta", Timestamp: "2023-01-01T00:00:00Z", FailingSyncSets: "syncset3"},
			},
		},
		// Test sorting by failingSyncSets in descending order
		{
			name:      "sort_by_failingSyncSets_descending_order",
			sortField: "failingsyncsets",
			sortOrder: "desc",
			expected: []failingClusterSync{
				{Name: "zeta", Timestamp: "2023-01-01T00:00:00Z", FailingSyncSets: "syncset3"},
				{Name: "beta", Timestamp: "2023-02-01T00:00:00Z", FailingSyncSets: "syncset2"},
				{Name: "alpha", Timestamp: "2022-01-01T00:00:00Z", FailingSyncSets: "syncset1"},
			},
		},
		// Test invalid sort field
		{
			name:      "invalid_sort_order",
			sortField: "invalid",
			sortOrder: "asc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &clusterSyncFailuresOptions{
				sortField: tt.sortField,
				sortOrder: tt.sortOrder,
			}

			failingClusterSyncList := []failingClusterSync{
				{Name: "zeta", Timestamp: "2023-01-01T00:00:00Z", FailingSyncSets: "syncset3"},
				{Name: "alpha", Timestamp: "2022-01-01T00:00:00Z", FailingSyncSets: "syncset1"},
				{Name: "beta", Timestamp: "2023-02-01T00:00:00Z", FailingSyncSets: "syncset2"},
			}

			err := options.sortBy(failingClusterSyncList)
			if tt.name == "invalid_sort_order" {
				assert.Error(t, err)
				assert.Equal(t, "Specify one of the following fields as a sort argument: name, timestamp, failingsyncsets.", err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, failingClusterSyncList)
			}
		})
	}
}
