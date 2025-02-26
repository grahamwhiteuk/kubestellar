/*
Copyright 2022 The KubeStellar Authors.

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
	"fmt"
	"sync"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"

	edgev2alpha1 "github.com/kubestellar/kubestellar/pkg/apis/edge/v2alpha1"
	"github.com/kubestellar/kubestellar/pkg/syncer/clientfactory"
)

const (
	DOWNSYNC_NAMESPACED_SUFFIX         string = "_downsync_namespaced"
	DOWNSYNC_NAMESPACED_OBJECTS_SUFFIX string = "_downsync_namespaced_objects"
	DOWNSYNC_CLUSTERSCOPED_SUFFIX      string = "_downsync_clusterscoped"
	UPSYNC_SUFFIX                      string = "_upsync"
)

func NewSyncerConfigManager(logger klog.Logger, syncConfigManager *SyncConfigManager, upstreamClientFactory clientfactory.ClientFactory, downstreamClientFactory clientfactory.ClientFactory) *SyncerConfigManager {
	return &SyncerConfigManager{
		logger:                  logger,
		syncConfigManager:       syncConfigManager,
		syncerConfigMap:         map[string]edgev2alpha1.SyncerConfig{},
		upstreamClientFactory:   upstreamClientFactory,
		downstreamClientFactory: downstreamClientFactory,
	}
}

type SyncerConfigManager struct {
	sync.Mutex
	logger                  klog.Logger
	syncConfigManager       *SyncConfigManager
	syncerConfigMap         map[string]edgev2alpha1.SyncerConfig
	upstreamClientFactory   clientfactory.ClientFactory
	downstreamClientFactory clientfactory.ClientFactory
}

func (s *SyncerConfigManager) upsert(syncerConfig edgev2alpha1.SyncerConfig) {
	logger := s.logger.WithValues("syncerConfigName", syncerConfig.Name)
	s.Lock()
	defer s.Unlock()
	s.syncerConfigMap[syncerConfig.Name] = syncerConfig
	logger.V(3).Info("upsert syncerConfig")
}

func (s *SyncerConfigManager) Refresh() {
	s.Lock()
	defer s.Unlock()
	for _, syncerConfig := range s.syncerConfigMap {
		logger := s.logger.WithValues("syncerConfigName", syncerConfig.Name)
		logger.V(3).Info("upsert syncerConfig to syncConfigManager stores")
		upstreamGroupResourcesList, err := s.upstreamClientFactory.GetAPIGroupResources()
		if err != nil {
			logger.Error(err, "Failed to get API Group resources from upstream. Skip upsert operation")
			return
		}
		downstreamGroupResourcesList, err := s.downstreamClientFactory.GetAPIGroupResources()
		if err != nil {
			logger.Error(err, "Failed to get API Group resources from downstream. Skip upsert operation")
			return
		}
		s.upsertNamespacedObjects(syncerConfig, upstreamGroupResourcesList)
		s.upsertNamespaceScoped(syncerConfig, upstreamGroupResourcesList)
		s.upsertClusterScoped(syncerConfig, upstreamGroupResourcesList)
		s.upsertUpsync(syncerConfig, downstreamGroupResourcesList)
	}
}

func (s *SyncerConfigManager) upsertNamespaceScoped(syncerConfig edgev2alpha1.SyncerConfig, upstreamGroupResourcesList []*restmapper.APIGroupResources) {
	s.logger.V(3).Info("upsert namespace scoped resources as syncerConfig to syncConfigManager stores", "syncerConfigName", syncerConfig.Name, "numNamespaces", len(syncerConfig.Spec.NamespaceScope.Namespaces))
	if lgr := s.logger.V(4); lgr.Enabled() {
		for _, agrs := range upstreamGroupResourcesList {
			lgr.Info("APIGroupResources", "group", agrs.Group, "versionedResources", agrs.VersionedResources)
		}
	}
	edgeSyncConfigResources := []edgev2alpha1.EdgeSyncConfigResource{}
	for _, namespace := range syncerConfig.Spec.NamespaceScope.Namespaces {
		edgeSyncConfigResourceForNamespace := edgev2alpha1.EdgeSyncConfigResource{
			Group:   "",
			Version: "v1",
			Kind:    "Namespace",
			Name:    namespace,
		}
		edgeSyncConfigResources = append(edgeSyncConfigResources, edgeSyncConfigResourceForNamespace)
		for _, syncerConfigResource := range syncerConfig.Spec.NamespaceScope.Resources {
			group := syncerConfigResource.Group
			version := syncerConfigResource.APIVersion
			resource := syncerConfigResource.Resource
			versionedResources := findVersionedResourcesByGVR(group, version, resource, upstreamGroupResourcesList, s.logger)
			s.logger.V(4).Info("Mapped GVR to GVKs", "namespace", namespace, "gvr", syncerConfigResource, "gvks", versionedResources)
			for _, versionedResource := range versionedResources {
				edgeSyncConfigResource := edgev2alpha1.EdgeSyncConfigResource{
					Group:     group,
					Version:   version,
					Kind:      versionedResource.Kind,
					Namespace: namespace,
					Name:      "*",
				}
				edgeSyncConfigResources = append(edgeSyncConfigResources, edgeSyncConfigResource)
			}
		}
	}
	edgeSyncConfig := edgev2alpha1.EdgeSyncConfig{
		ObjectMeta: v1.ObjectMeta{
			Name: syncerConfig.Name + DOWNSYNC_NAMESPACED_SUFFIX,
		},
		Spec: edgev2alpha1.EdgeSyncConfigSpec{
			DownSyncedResources: edgeSyncConfigResources,
		},
	}
	s.syncConfigManager.upsert(edgeSyncConfig)
}

type flatNamespacedObject struct {
	APIVersion string
	v1.GroupResource
	Namespace string
	Names     []string
}

type tableOfFlatNamespacedObject struct {
	FlatNamespacedObjects []flatNamespacedObject
}

func (t *tableOfFlatNamespacedObject) filter(predicate func(row flatNamespacedObject) bool) *tableOfFlatNamespacedObject {
	flatNamespacedObjects := []flatNamespacedObject{}
	for _, obj := range t.FlatNamespacedObjects {
		if predicate(obj) {
			flatNamespacedObjects = append(flatNamespacedObjects, obj)
		}
	}
	return &tableOfFlatNamespacedObject{FlatNamespacedObjects: flatNamespacedObjects}
}

func (t *tableOfFlatNamespacedObject) getAllNamespaces() []string {
	namespaces := sets.String{}
	for _, obj := range t.FlatNamespacedObjects {
		namespaces.Insert(obj.Namespace)
	}
	return namespaces.List()
}

func (s *SyncerConfigManager) upsertNamespacedObjects(syncerConfig edgev2alpha1.SyncerConfig, upstreamGroupResourcesList []*restmapper.APIGroupResources) {
	flatNamespacedObjects := []flatNamespacedObject{}
	for _, nsObject := range syncerConfig.Spec.NamespacedObjects {
		for _, obj := range nsObject.ObjectsByNamespace {
			flatNamespacedObjects = append(flatNamespacedObjects, flatNamespacedObject{
				APIVersion:    nsObject.APIVersion,
				GroupResource: nsObject.GroupResource,
				Namespace:     obj.Namespace,
				Names:         obj.Names,
			})
		}
	}
	namespacedObjectsTable := tableOfFlatNamespacedObject{FlatNamespacedObjects: flatNamespacedObjects}

	requiredNamespaces := namespacedObjectsTable.filter(func(row flatNamespacedObject) bool { return len(row.Names) > 0 }).getAllNamespaces()

	s.logger.V(3).Info("upsert namespaced objects as syncerConfig to syncConfigManager stores", "syncerConfigName", syncerConfig.Name, "numNamespaces", len(requiredNamespaces))
	if lgr := s.logger.V(4); lgr.Enabled() {
		for _, agrs := range upstreamGroupResourcesList {
			lgr.Info("APIGroupResources", "group", agrs.Group, "versionedResources", agrs.VersionedResources)
		}
	}
	edgeSyncConfigResources := []edgev2alpha1.EdgeSyncConfigResource{}
	for _, namespace := range requiredNamespaces {
		edgeSyncConfigResourceForNamespace := edgev2alpha1.EdgeSyncConfigResource{
			Group:   "",
			Version: "v1",
			Kind:    "Namespace",
			Name:    namespace,
		}
		edgeSyncConfigResources = append(edgeSyncConfigResources, edgeSyncConfigResourceForNamespace)
	}

	for _, object := range namespacedObjectsTable.FlatNamespacedObjects {
		group := object.Group
		version := object.APIVersion
		resource := object.Resource
		gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
		versionedResources := findVersionedResourcesByGVR(group, version, resource, upstreamGroupResourcesList, s.logger)
		s.logger.V(4).Info("Mapped GVR to GVKs", "namespace", object.Namespace, "gvr", gvr, "gvks", versionedResources)
		for _, versionedResource := range versionedResources {
			for _, name := range object.Names {
				edgeSyncConfigResource := edgev2alpha1.EdgeSyncConfigResource{
					Group:     group,
					Version:   version,
					Kind:      versionedResource.Kind,
					Namespace: object.Namespace,
					Name:      name,
				}
				edgeSyncConfigResources = append(edgeSyncConfigResources, edgeSyncConfigResource)
			}
		}
	}
	edgeSyncConfig := edgev2alpha1.EdgeSyncConfig{
		ObjectMeta: v1.ObjectMeta{
			Name: syncerConfig.Name + DOWNSYNC_NAMESPACED_OBJECTS_SUFFIX,
		},
		Spec: edgev2alpha1.EdgeSyncConfigSpec{
			DownSyncedResources: edgeSyncConfigResources,
		},
	}
	s.syncConfigManager.upsert(edgeSyncConfig)
}

func (s *SyncerConfigManager) upsertClusterScoped(syncerConfig edgev2alpha1.SyncerConfig, upstreamGroupResourcesList []*restmapper.APIGroupResources) {
	s.logger.V(3).Info(fmt.Sprintf("upsert clusterscoped resources as syncerConfig %s to syncConfigManager stores", syncerConfig.Name))
	edgeSyncConfigResources := []edgev2alpha1.EdgeSyncConfigResource{}
	for _, clusterScope := range syncerConfig.Spec.ClusterScope {
		group := clusterScope.Group
		version := clusterScope.APIVersion
		resource := clusterScope.Resource
		objects := clusterScope.Objects
		if objects != nil && len(objects) == 0 {
			// empty list means nothing to downsync (see ClusterScopeDownsyncResource definition)
			continue
		}
		versionedResources := findVersionedResourcesByGVR(group, version, resource, upstreamGroupResourcesList, s.logger)
		for _, versionedResource := range versionedResources {
			if objects == nil {
				edgeSyncConfigResource := edgev2alpha1.EdgeSyncConfigResource{
					Group:   group,
					Version: version,
					Kind:    versionedResource.Kind,
					Name:    "*",
				}
				edgeSyncConfigResources = append(edgeSyncConfigResources, edgeSyncConfigResource)
			} else {
				for _, object := range objects {
					edgeSyncConfigResource := edgev2alpha1.EdgeSyncConfigResource{
						Group:   group,
						Version: version,
						Kind:    versionedResource.Kind,
						Name:    object,
					}
					edgeSyncConfigResources = append(edgeSyncConfigResources, edgeSyncConfigResource)
				}
			}
		}
	}
	edgeSyncConfig := edgev2alpha1.EdgeSyncConfig{
		ObjectMeta: v1.ObjectMeta{
			Name: syncerConfig.Name + DOWNSYNC_CLUSTERSCOPED_SUFFIX,
		},
		Spec: edgev2alpha1.EdgeSyncConfigSpec{
			DownSyncedResources: edgeSyncConfigResources,
		},
	}
	s.syncConfigManager.upsert(edgeSyncConfig)
}

func (s *SyncerConfigManager) upsertUpsync(syncerConfig edgev2alpha1.SyncerConfig, downstreamGroupResourcesList []*restmapper.APIGroupResources) {
	s.logger.V(3).Info(fmt.Sprintf("upsert upsynced resources as syncerConfig %s to syncConfigManager stores", syncerConfig.Name))
	edgeSyncConfigResources := []edgev2alpha1.EdgeSyncConfigResource{}
	upsyncedNamespaces := sets.String{}
	for _, upsync := range syncerConfig.Spec.Upsync {
		upsyncedNamespaces.Insert(upsync.Namespaces...)
	}
	for _, namespace := range upsyncedNamespaces.List() {
		edgeSyncConfigResource := edgev2alpha1.EdgeSyncConfigResource{
			Group:   "",
			Version: "v1",
			Kind:    "Namespace",
			Name:    namespace,
		}
		edgeSyncConfigResources = append(edgeSyncConfigResources, edgeSyncConfigResource)
	}
	for _, upsync := range syncerConfig.Spec.Upsync {
		group := upsync.APIGroup
		resources := upsync.Resources
		namespaces := upsync.Namespaces
		names := upsync.Names
		for _, resource := range resources {
			versionedResources := findVersionedResourcesByGV(group, resource, downstreamGroupResourcesList, s.logger)
			for _, versionedResource := range versionedResources {
				edgeSyncConfigResource := edgev2alpha1.EdgeSyncConfigResource{
					Group:   group,
					Version: versionedResource.Version,
					Kind:    versionedResource.Kind,
				}
				if versionedResource.Namespaced {
					for _, namespace := range namespaces {
						for _, name := range names {
							edgeSyncConfigResource.Namespace = namespace
							edgeSyncConfigResource.Name = name
							edgeSyncConfigResources = append(edgeSyncConfigResources, edgeSyncConfigResource)
						}
					}
				} else {
					for _, name := range names {
						edgeSyncConfigResource.Name = name
						edgeSyncConfigResources = append(edgeSyncConfigResources, edgeSyncConfigResource)
					}
				}
			}
		}
	}
	edgeSyncConfig := edgev2alpha1.EdgeSyncConfig{
		ObjectMeta: v1.ObjectMeta{
			Name: syncerConfig.Name + UPSYNC_SUFFIX,
		},
		Spec: edgev2alpha1.EdgeSyncConfigSpec{
			UpSyncedResources: edgeSyncConfigResources,
		},
	}
	s.syncConfigManager.upsert(edgeSyncConfig)
}

func (s *SyncerConfigManager) delete(key string) {
	logger := s.logger.WithValues("syncerConfigName", key)
	logger.V(3).Info("delete syncConfigs for syncerConfig from syncConfigManager stores")
	s.Lock()
	defer s.Unlock()
	delete(s.syncerConfigMap, key)
	s.syncConfigManager.delete(key + DOWNSYNC_NAMESPACED_SUFFIX)
	s.syncConfigManager.delete(key + DOWNSYNC_NAMESPACED_OBJECTS_SUFFIX)
	s.syncConfigManager.delete(key + DOWNSYNC_CLUSTERSCOPED_SUFFIX)
	s.syncConfigManager.delete(key + UPSYNC_SUFFIX)
}

func findVersionedResourcesByGVR(group string, version string, resource string, apiGroupResourcesList []*restmapper.APIGroupResources, logger klog.Logger) []v1.APIResource {
	_versionedResources := []v1.APIResource{}
	var apiGroupResources *restmapper.APIGroupResources
	for _, groupResources := range apiGroupResourcesList {
		if groupResources.Group.Name == group {
			apiGroupResources = groupResources
			break
		}
	}
	if apiGroupResources != nil {
		versionedResources := apiGroupResources.VersionedResources[version]
		for _, versionedResource := range versionedResources {
			if resource == versionedResource.Name {
				_versionedResources = append(_versionedResources, versionedResource)
			} else if resource == "*" {
				_versionedResources = append(_versionedResources, versionedResource)
			}
		}
	}
	if len(_versionedResources) == 0 {
		logger.V(2).Info("Any versioned resource is not found from given apiGroupResources", "group", group, "version", version, "resource", resource)
	}
	return _versionedResources
}

func findVersionedResourcesByGV(group string, resource string, apiGroupResourcesList []*restmapper.APIGroupResources, logger klog.Logger) []v1.APIResource {
	_versionedResources := []v1.APIResource{}
	var apiGroupResources *restmapper.APIGroupResources
	for _, groupResources := range apiGroupResourcesList {
		if groupResources.Group.Name == group {
			apiGroupResources = groupResources
			break
		}
	}
	if apiGroupResources == nil {
		return _versionedResources
	}
	for version, versionedResources := range apiGroupResources.VersionedResources {
		for _, versionedResource := range versionedResources {
			if versionedResource.Version == "" {
				versionedResource.Version = version
			}
			if resource == versionedResource.Name {
				_versionedResources = append(_versionedResources, versionedResource)
			} else if resource == "*" {
				_versionedResources = append(_versionedResources, versionedResource)
			}
		}
	}
	return _versionedResources
}
