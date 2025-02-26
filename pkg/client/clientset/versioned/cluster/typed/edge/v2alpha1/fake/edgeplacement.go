//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright The KubeStellar Authors.

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

// Code generated by kcp code-generator. DO NOT EDIT.

package v2alpha1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/testing"

	kcptesting "github.com/kcp-dev/client-go/third_party/k8s.io/client-go/testing"
	"github.com/kcp-dev/logicalcluster/v3"

	edgev2alpha1 "github.com/kubestellar/kubestellar/pkg/apis/edge/v2alpha1"
	edgev2alpha1client "github.com/kubestellar/kubestellar/pkg/client/clientset/versioned/typed/edge/v2alpha1"
)

var edgePlacementsResource = schema.GroupVersionResource{Group: "edge.kubestellar.io", Version: "v2alpha1", Resource: "edgeplacements"}
var edgePlacementsKind = schema.GroupVersionKind{Group: "edge.kubestellar.io", Version: "v2alpha1", Kind: "EdgePlacement"}

type edgePlacementsClusterClient struct {
	*kcptesting.Fake
}

// Cluster scopes the client down to a particular cluster.
func (c *edgePlacementsClusterClient) Cluster(clusterPath logicalcluster.Path) edgev2alpha1client.EdgePlacementInterface {
	if clusterPath == logicalcluster.Wildcard {
		panic("A specific cluster must be provided when scoping, not the wildcard.")
	}

	return &edgePlacementsClient{Fake: c.Fake, ClusterPath: clusterPath}
}

// List takes label and field selectors, and returns the list of EdgePlacements that match those selectors across all clusters.
func (c *edgePlacementsClusterClient) List(ctx context.Context, opts metav1.ListOptions) (*edgev2alpha1.EdgePlacementList, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootListAction(edgePlacementsResource, edgePlacementsKind, logicalcluster.Wildcard, opts), &edgev2alpha1.EdgePlacementList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &edgev2alpha1.EdgePlacementList{ListMeta: obj.(*edgev2alpha1.EdgePlacementList).ListMeta}
	for _, item := range obj.(*edgev2alpha1.EdgePlacementList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested EdgePlacements across all clusters.
func (c *edgePlacementsClusterClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.InvokesWatch(kcptesting.NewRootWatchAction(edgePlacementsResource, logicalcluster.Wildcard, opts))
}

type edgePlacementsClient struct {
	*kcptesting.Fake
	ClusterPath logicalcluster.Path
}

func (c *edgePlacementsClient) Create(ctx context.Context, edgePlacement *edgev2alpha1.EdgePlacement, opts metav1.CreateOptions) (*edgev2alpha1.EdgePlacement, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootCreateAction(edgePlacementsResource, c.ClusterPath, edgePlacement), &edgev2alpha1.EdgePlacement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*edgev2alpha1.EdgePlacement), err
}

func (c *edgePlacementsClient) Update(ctx context.Context, edgePlacement *edgev2alpha1.EdgePlacement, opts metav1.UpdateOptions) (*edgev2alpha1.EdgePlacement, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootUpdateAction(edgePlacementsResource, c.ClusterPath, edgePlacement), &edgev2alpha1.EdgePlacement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*edgev2alpha1.EdgePlacement), err
}

func (c *edgePlacementsClient) UpdateStatus(ctx context.Context, edgePlacement *edgev2alpha1.EdgePlacement, opts metav1.UpdateOptions) (*edgev2alpha1.EdgePlacement, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootUpdateSubresourceAction(edgePlacementsResource, c.ClusterPath, "status", edgePlacement), &edgev2alpha1.EdgePlacement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*edgev2alpha1.EdgePlacement), err
}

func (c *edgePlacementsClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.Invokes(kcptesting.NewRootDeleteActionWithOptions(edgePlacementsResource, c.ClusterPath, name, opts), &edgev2alpha1.EdgePlacement{})
	return err
}

func (c *edgePlacementsClient) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := kcptesting.NewRootDeleteCollectionAction(edgePlacementsResource, c.ClusterPath, listOpts)

	_, err := c.Fake.Invokes(action, &edgev2alpha1.EdgePlacementList{})
	return err
}

func (c *edgePlacementsClient) Get(ctx context.Context, name string, options metav1.GetOptions) (*edgev2alpha1.EdgePlacement, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootGetAction(edgePlacementsResource, c.ClusterPath, name), &edgev2alpha1.EdgePlacement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*edgev2alpha1.EdgePlacement), err
}

// List takes label and field selectors, and returns the list of EdgePlacements that match those selectors.
func (c *edgePlacementsClient) List(ctx context.Context, opts metav1.ListOptions) (*edgev2alpha1.EdgePlacementList, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootListAction(edgePlacementsResource, edgePlacementsKind, c.ClusterPath, opts), &edgev2alpha1.EdgePlacementList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &edgev2alpha1.EdgePlacementList{ListMeta: obj.(*edgev2alpha1.EdgePlacementList).ListMeta}
	for _, item := range obj.(*edgev2alpha1.EdgePlacementList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

func (c *edgePlacementsClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.InvokesWatch(kcptesting.NewRootWatchAction(edgePlacementsResource, c.ClusterPath, opts))
}

func (c *edgePlacementsClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (*edgev2alpha1.EdgePlacement, error) {
	obj, err := c.Fake.Invokes(kcptesting.NewRootPatchSubresourceAction(edgePlacementsResource, c.ClusterPath, name, pt, data, subresources...), &edgev2alpha1.EdgePlacement{})
	if obj == nil {
		return nil, err
	}
	return obj.(*edgev2alpha1.EdgePlacement), err
}
