/*
Copyright The Kubernetes Authors.

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
	controllerv1 "github.com/chenleji/pipeline-operator/api/controller/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakePipelines implements PipelineInterface
type FakePipelines struct {
	Fake *FakeControllerV1
	ns   string
}

var pipelinesResource = schema.GroupVersionResource{Group: "controller.ljchen.net", Version: "v1", Resource: "pipelines"}

var pipelinesKind = schema.GroupVersionKind{Group: "controller.ljchen.net", Version: "v1", Kind: "Pipeline"}

// Get takes name of the pipeline, and returns the corresponding pipeline object, and an error if there is any.
func (c *FakePipelines) Get(name string, options v1.GetOptions) (result *controllerv1.Pipeline, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(pipelinesResource, c.ns, name), &controllerv1.Pipeline{})

	if obj == nil {
		return nil, err
	}
	return obj.(*controllerv1.Pipeline), err
}

// List takes label and field selectors, and returns the list of Pipelines that match those selectors.
func (c *FakePipelines) List(opts v1.ListOptions) (result *controllerv1.PipelineList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(pipelinesResource, pipelinesKind, c.ns, opts), &controllerv1.PipelineList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &controllerv1.PipelineList{ListMeta: obj.(*controllerv1.PipelineList).ListMeta}
	for _, item := range obj.(*controllerv1.PipelineList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested pipelines.
func (c *FakePipelines) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(pipelinesResource, c.ns, opts))

}

// Create takes the representation of a pipeline and creates it.  Returns the server's representation of the pipeline, and an error, if there is any.
func (c *FakePipelines) Create(pipeline *controllerv1.Pipeline) (result *controllerv1.Pipeline, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(pipelinesResource, c.ns, pipeline), &controllerv1.Pipeline{})

	if obj == nil {
		return nil, err
	}
	return obj.(*controllerv1.Pipeline), err
}

// Update takes the representation of a pipeline and updates it. Returns the server's representation of the pipeline, and an error, if there is any.
func (c *FakePipelines) Update(pipeline *controllerv1.Pipeline) (result *controllerv1.Pipeline, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(pipelinesResource, c.ns, pipeline), &controllerv1.Pipeline{})

	if obj == nil {
		return nil, err
	}
	return obj.(*controllerv1.Pipeline), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakePipelines) UpdateStatus(pipeline *controllerv1.Pipeline) (*controllerv1.Pipeline, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(pipelinesResource, "status", c.ns, pipeline), &controllerv1.Pipeline{})

	if obj == nil {
		return nil, err
	}
	return obj.(*controllerv1.Pipeline), err
}

// Delete takes name of the pipeline and deletes it. Returns an error if one occurs.
func (c *FakePipelines) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(pipelinesResource, c.ns, name), &controllerv1.Pipeline{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakePipelines) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(pipelinesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &controllerv1.PipelineList{})
	return err
}

// Patch applies the patch and returns the patched pipeline.
func (c *FakePipelines) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *controllerv1.Pipeline, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(pipelinesResource, c.ns, name, pt, data, subresources...), &controllerv1.Pipeline{})

	if obj == nil {
		return nil, err
	}
	return obj.(*controllerv1.Pipeline), err
}
