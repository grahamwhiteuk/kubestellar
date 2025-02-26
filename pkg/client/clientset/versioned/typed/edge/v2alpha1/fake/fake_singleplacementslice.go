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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"

	v2alpha1 "github.com/kubestellar/kubestellar/pkg/apis/edge/v2alpha1"
)

// FakeSinglePlacementSlices implements SinglePlacementSliceInterface
type FakeSinglePlacementSlices struct {
	Fake *FakeEdgeV2alpha1
}

var singleplacementslicesResource = schema.GroupVersionResource{Group: "edge.kubestellar.io", Version: "v2alpha1", Resource: "singleplacementslices"}

var singleplacementslicesKind = schema.GroupVersionKind{Group: "edge.kubestellar.io", Version: "v2alpha1", Kind: "SinglePlacementSlice"}

// Get takes name of the singlePlacementSlice, and returns the corresponding singlePlacementSlice object, and an error if there is any.
func (c *FakeSinglePlacementSlices) Get(ctx context.Context, name string, options v1.GetOptions) (result *v2alpha1.SinglePlacementSlice, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(singleplacementslicesResource, name), &v2alpha1.SinglePlacementSlice{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v2alpha1.SinglePlacementSlice), err
}

// List takes label and field selectors, and returns the list of SinglePlacementSlices that match those selectors.
func (c *FakeSinglePlacementSlices) List(ctx context.Context, opts v1.ListOptions) (result *v2alpha1.SinglePlacementSliceList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(singleplacementslicesResource, singleplacementslicesKind, opts), &v2alpha1.SinglePlacementSliceList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v2alpha1.SinglePlacementSliceList{ListMeta: obj.(*v2alpha1.SinglePlacementSliceList).ListMeta}
	for _, item := range obj.(*v2alpha1.SinglePlacementSliceList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested singlePlacementSlices.
func (c *FakeSinglePlacementSlices) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(singleplacementslicesResource, opts))
}

// Create takes the representation of a singlePlacementSlice and creates it.  Returns the server's representation of the singlePlacementSlice, and an error, if there is any.
func (c *FakeSinglePlacementSlices) Create(ctx context.Context, singlePlacementSlice *v2alpha1.SinglePlacementSlice, opts v1.CreateOptions) (result *v2alpha1.SinglePlacementSlice, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(singleplacementslicesResource, singlePlacementSlice), &v2alpha1.SinglePlacementSlice{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v2alpha1.SinglePlacementSlice), err
}

// Update takes the representation of a singlePlacementSlice and updates it. Returns the server's representation of the singlePlacementSlice, and an error, if there is any.
func (c *FakeSinglePlacementSlices) Update(ctx context.Context, singlePlacementSlice *v2alpha1.SinglePlacementSlice, opts v1.UpdateOptions) (result *v2alpha1.SinglePlacementSlice, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(singleplacementslicesResource, singlePlacementSlice), &v2alpha1.SinglePlacementSlice{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v2alpha1.SinglePlacementSlice), err
}

// Delete takes name of the singlePlacementSlice and deletes it. Returns an error if one occurs.
func (c *FakeSinglePlacementSlices) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(singleplacementslicesResource, name, opts), &v2alpha1.SinglePlacementSlice{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeSinglePlacementSlices) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(singleplacementslicesResource, listOpts)

	_, err := c.Fake.Invokes(action, &v2alpha1.SinglePlacementSliceList{})
	return err
}

// Patch applies the patch and returns the patched singlePlacementSlice.
func (c *FakeSinglePlacementSlices) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v2alpha1.SinglePlacementSlice, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(singleplacementslicesResource, name, pt, data, subresources...), &v2alpha1.SinglePlacementSlice{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v2alpha1.SinglePlacementSlice), err
}
