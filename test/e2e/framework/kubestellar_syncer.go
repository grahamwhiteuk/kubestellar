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

package framework

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	kubernetesclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/yaml"

	workloadcliplugin "github.com/kubestellar/kubestellar/pkg/cliplugins/kubestellar/syncer-gen"
	"github.com/kubestellar/kubestellar/pkg/syncer"
	"github.com/kubestellar/kubestellar/test/e2e/logicalcluster"
)

//go:embed testdata/*
var embedded embed.FS

var crdGVR = schema.GroupVersionResource{
	Group:    "apiextensions.k8s.io",
	Version:  "v1",
	Resource: "customresourcedefinitions",
}

var clusterroleGVR = schema.GroupVersionResource{
	Group:    "rbac.authorization.k8s.io",
	Version:  "v1",
	Resource: "clusterroles",
}

var clusterrolebindingGVR = schema.GroupVersionResource{
	Group:    "rbac.authorization.k8s.io",
	Version:  "v1",
	Resource: "clusterrolebindings",
}

var apibindingGVR = schema.GroupVersionResource{
	Group:    "apis.kcp.io",
	Version:  "v1alpha1",
	Resource: "apibindings",
}

var edgeSyncConfigGVR = schema.GroupVersionResource{
	Group:    "edge.kubestellar.io",
	Version:  "v2alpha1",
	Resource: "edgesyncconfigs",
}

var syncerConfigGVR = schema.GroupVersionResource{
	Group:    "edge.kubestellar.io",
	Version:  "v2alpha1",
	Resource: "syncerconfigs",
}

type SyncerOption func(t *testing.T, fs *kubeStellarSyncerFixture)

func NewKubeStellarSyncerFixture(t *testing.T, server *kcpServer, path logicalcluster.Path) *kubeStellarSyncerFixture {
	t.Helper()

	sf := &kubeStellarSyncerFixture{
		upstreamServer:     server,
		edgeSyncTargetPath: path,
		edgeSyncTargetName: "psyncer-01",
	}
	return sf
}

// kubeStellarSyncerFixture configures a syncer fixture. Its `Start` method does the work of starting a syncer.
type kubeStellarSyncerFixture struct {
	upstreamServer     *kcpServer
	edgeSyncTargetPath logicalcluster.Path
	edgeSyncTargetName string
}

// CreateEdgeSyncTargetAndApplyToDownstream creates a default EdgeSyncConfig resource through the `kubestellar syncer-gen` CLI command,
// applies the kubestellar-syncer-related resources in the WEC.
func (sf *kubeStellarSyncerFixture) CreateEdgeSyncTargetAndApplyToDownstream(t *testing.T) *appliedKubeStellarSyncerFixture {
	t.Helper()

	// Write the upstream logical cluster config to disk for the workspace plugin
	upstreamRawConfig, err := sf.upstreamServer.RawConfig()
	require.NoError(t, err)
	_, kubeconfigPath := WriteLogicalClusterConfig(t, upstreamRawConfig, "base", sf.edgeSyncTargetPath)

	var downstreamConfig *rest.Config
	var downstreamKubeconfigPath string

	// The syncer will target a logical cluster that is a child of the current workspace. A
	// logical server provides as a lightweight approximation of a WEC for tests that
	// don't need to validate running workloads or interaction with kube controllers.
	downstreamServer := NewFakeWorkloadServer(t, sf.upstreamServer, sf.edgeSyncTargetPath, sf.edgeSyncTargetName)
	downstreamConfig = downstreamServer.BaseConfig(t)
	downstreamKubeconfigPath = downstreamServer.KubeconfigPath()
	syncerImage := "not-a-valid-image"

	// Modify root:compute so that Syncer can update deployment.status
	modifyRootCompute(t, upstreamRawConfig)

	downstreamKubeClient, err := kubernetesclient.NewForConfig(downstreamConfig)
	require.NoError(t, err)
	downstreamDynamicKubeClient, err := dynamic.NewForConfig(downstreamConfig)
	require.NoError(t, err)

	logicalConfig, upstreamKubeconfigPath := WriteLogicalClusterConfig(t, upstreamRawConfig, "base", sf.edgeSyncTargetPath)
	upstreamKubeConfig, err := logicalConfig.ClientConfig()
	require.NoError(t, err)
	upstreamKubeClient, err := kubernetesclient.NewForConfig(upstreamKubeConfig)
	require.NoError(t, err)
	upstreamDynamicKubeClient, err := dynamic.NewForConfig(upstreamKubeConfig)
	require.NoError(t, err)

	var syncerConfigCRDUnst *unstructured.Unstructured
	err = LoadFile(repositoryDir()+"/config/crds/edge.kubestellar.io_syncerconfigs.yaml", &osReader{}, &syncerConfigCRDUnst)
	require.NoError(t, err)
	t.Logf("Create SyncerConfig CRD in workspace %q.", sf.edgeSyncTargetPath)
	_, err = upstreamDynamicKubeClient.Resource(crdGVR).Create(context.Background(), syncerConfigCRDUnst, v1.CreateOptions{})
	require.NoError(t, err)

	var edgeSyncConfigCRDUnst *unstructured.Unstructured
	err = LoadFile(repositoryDir()+"/config/crds/edge.kubestellar.io_edgesyncconfigs.yaml", &osReader{}, &edgeSyncConfigCRDUnst)
	require.NoError(t, err)
	t.Logf("Create EdgeSyncConfig CRD in workspace %q.", sf.edgeSyncTargetPath)
	_, err = upstreamDynamicKubeClient.Resource(crdGVR).Create(context.Background(), edgeSyncConfigCRDUnst, v1.CreateOptions{})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, err = upstreamDynamicKubeClient.Resource(edgeSyncConfigGVR).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Logf("error seen waiting for EdgeSyncConfig API to be available: %v", err)
			return false
		}
		_, err = upstreamDynamicKubeClient.Resource(syncerConfigGVR).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Logf("error seen waiting for SyncerConfig API to be available: %v", err)
			return false
		}
		return true
	}, wait.ForeverTestTimeout, time.Millisecond*100)

	sf.createComputeResources(t, upstreamKubeClient, upstreamDynamicKubeClient)
	sf.createComputeResources(t, downstreamKubeClient, downstreamDynamicKubeClient)

	// Run the plugin command to enable the kubestellar syncer and collect the resulting yaml
	t.Logf("Configuring workspace %s for syncing", sf.edgeSyncTargetPath)
	pluginArgs := []string{
		sf.edgeSyncTargetName,
		"--syncer-image=" + syncerImage,
		"--output-file=-",
	}

	syncerYAML := RunKcpEdgeCliPlugin(t, kubeconfigPath, pluginArgs)

	// Apply the yaml output from the plugin to the downstream server
	KubectlApply(t, downstreamKubeconfigPath, syncerYAML)

	// Extract the configuration for an in-process syncer from the resources that were
	// applied to the downstream server. This maximizes the parity between the
	// configuration of a deployed and in-process syncer.
	var syncerID string
	for _, doc := range strings.Split(string(syncerYAML), "\n---\n") {
		var manifest struct {
			metav1.ObjectMeta `json:"metadata"`
		}
		err := yaml.Unmarshal([]byte(doc), &manifest)
		require.NoError(t, err)
		if manifest.Namespace != "" {
			syncerID = manifest.Namespace
			break
		}
	}
	require.NotEmpty(t, syncerID, "failed to extract syncer namespace from yaml produced by plugin:\n%s", string(syncerYAML))

	syncerConfig := syncerConfigFromCluster(t, downstreamConfig, syncerID, syncerID)

	return &appliedKubeStellarSyncerFixture{
		kubeStellarSyncerFixture: *sf,

		SyncerConfig:                syncerConfig,
		SyncerID:                    syncerID,
		WorkspacePath:               sf.edgeSyncTargetPath,
		DownstreamConfig:            downstreamConfig,
		DownstreamKubeClient:        downstreamKubeClient,
		DownstreamDynamicKubeClient: downstreamDynamicKubeClient,
		DownstreamKubeconfigPath:    downstreamKubeconfigPath,
		UpstreamConfig:              upstreamKubeConfig,
		UpstreamKubeClusterClient:   &KcpClusterClient{client: upstreamKubeClient},
		UpstreamDynamicKubeClient:   &KcpDynamicClient{client: upstreamDynamicKubeClient},
		UpstreamKubeconfigPath:      upstreamKubeconfigPath,
	}
}

func (sf *kubeStellarSyncerFixture) createComputeResources(t *testing.T, client *kubernetesclient.Clientset, dynamicClient dynamic.Interface) {
	var apibindingUnst *unstructured.Unstructured
	err := LoadFile("testdata/apibinding.yaml", embedded, &apibindingUnst)
	require.NoError(t, err)
	t.Log("Create apibinding (root:compute:kubernetes) in workspace.")
	_, err = dynamicClient.Resource(apibindingGVR).Create(context.Background(), apibindingUnst, v1.CreateOptions{})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, err := client.AppsV1().Deployments("").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Logf("error seen waiting for deployment crd to become active: %v", err)
			return false
		}
		_, err = client.CoreV1().Services("").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Logf("error seen waiting for service crd to become active: %v", err)
			return false
		}
		_, err = client.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Logf("error seen waiting for pods crd to become active: %v", err)
			return false
		}
		return true
	}, wait.ForeverTestTimeout, time.Millisecond*100)
}

// RunSyncer runs a new Syncer against the upstream kcp workspaces
// Whether the syncer runs in-process or deployed on a WEC will depend
// whether --wec-kubeconfig and --syncer-image are supplied to the test invocation.
func (sf *appliedKubeStellarSyncerFixture) RunSyncer(t *testing.T) *StartedKubeStellarSyncerFixture {
	t.Helper()

	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		sf.SyncerConfig.DownstreamConfig.Burst = 128
		sf.SyncerConfig.DownstreamConfig.QPS = 128
		sf.SyncerConfig.UpstreamConfig.Burst = 128
		sf.SyncerConfig.UpstreamConfig.QPS = 128
		err := syncer.RunSyncer(ctx, sf.SyncerConfig, 1)
		require.NoError(t, err, "syncer failed to start")
	}()

	t.Cleanup(cancelFunc)

	return &StartedKubeStellarSyncerFixture{
		sf,
	}
}

// appliedKubeStellarSyncerFixture contains the configuration required to start an kubestellar syncer and interact with its
// downstream cluster.
type appliedKubeStellarSyncerFixture struct {
	kubeStellarSyncerFixture

	SyncerConfig  *syncer.SyncerConfig
	SyncerID      string
	WorkspacePath logicalcluster.Path
	// Provide cluster-admin config and client for test purposes. The downstream config in
	// SyncerConfig will be less privileged.
	DownstreamConfig            *rest.Config
	DownstreamKubeClient        kubernetesclient.Interface
	DownstreamDynamicKubeClient dynamic.Interface
	DownstreamKubeconfigPath    string

	UpstreamConfig            *rest.Config
	UpstreamKubeClusterClient *KcpClusterClient
	UpstreamDynamicKubeClient *KcpDynamicClient
	UpstreamKubeconfigPath    string
}

// StartedKubeStellarSyncerFixture contains the configuration used to start a syncer and interact with its
// downstream cluster.
type StartedKubeStellarSyncerFixture struct {
	*appliedKubeStellarSyncerFixture
}

func (sf *StartedKubeStellarSyncerFixture) DeleteRootComputeAPIBinding(t *testing.T) {
	err := sf.UpstreamDynamicKubeClient.Cluster(sf.WorkspacePath).Resource(apibindingGVR).Delete(context.Background(), "kubernetes", v1.DeleteOptions{})
	require.NoError(t, err)
}

// syncerConfigFromCluster reads the configuration needed to start an in-process
// syncer from the resources applied to a cluster for a deployed syncer.
func syncerConfigFromCluster(t *testing.T, downstreamConfig *rest.Config, namespace, syncerID string) *syncer.SyncerConfig {
	t.Helper()

	ctx, cancelFunc := context.WithCancel(context.Background())
	t.Cleanup(cancelFunc)

	downstreamKubeClient, err := kubernetesclient.NewForConfig(downstreamConfig)
	require.NoError(t, err)

	// Read the upstream kubeconfig from the syncer secret
	secret, err := downstreamKubeClient.CoreV1().Secrets(namespace).Get(ctx, syncerID, metav1.GetOptions{})
	require.NoError(t, err)
	upstreamConfigBytes := secret.Data[workloadcliplugin.SyncerSecretConfigKey]
	require.NotEmpty(t, upstreamConfigBytes, "upstream config is required")
	upstreamConfig, err := clientcmd.RESTConfigFromKubeConfig(upstreamConfigBytes)
	require.NoError(t, err, "failed to load upstream config")

	// Read the downstream token from the deployment's service account secret
	var tokenSecret corev1.Secret
	Eventually(t, func() (bool, string) {
		secrets, err := downstreamKubeClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Errorf("failed to list secrets: %v", err)
			return false, fmt.Sprintf("failed to list secrets downstream: %v", err)
		}
		for _, secret := range secrets.Items {
			t.Logf("checking secret %s/%s for annotation %s=%s", secret.Namespace, secret.Name, corev1.ServiceAccountNameKey, syncerID)
			if secret.Annotations[corev1.ServiceAccountNameKey] == syncerID {
				tokenSecret = secret
				return len(secret.Data["token"]) > 0, fmt.Sprintf("token secret %s/%s for service account %s found", namespace, secret.Name, syncerID)
			}
		}
		return false, fmt.Sprintf("token secret for service account %s/%s not found", namespace, syncerID)
	}, wait.ForeverTestTimeout, time.Millisecond*100, "token secret in namespace %q for syncer service account %q not found", namespace, syncerID)
	token := tokenSecret.Data["token"]
	require.NotEmpty(t, token, "token is required")

	// Compose a new downstream config that uses the token
	downstreamConfigWithToken := ConfigWithToken(string(token), rest.CopyConfig(downstreamConfig))
	return &syncer.SyncerConfig{
		UpstreamConfig:   upstreamConfig,
		DownstreamConfig: downstreamConfigWithToken,
		SyncTargetName:   "",
		SyncTargetUID:    "",
		Interval:         time.Second * 3,
	}
}

func modifyRootCompute(t *testing.T, upstreamRawConfig clientcmdapi.Config) {
	// Write the upstream root:compute logical cluster config to disk for the workspace plugin
	rootComputeClientConfig, _ := WriteLogicalClusterConfig(t, upstreamRawConfig, "base", logicalcluster.NewPath("root:compute"))
	rootComputeKubeconfig, err := rootComputeClientConfig.ClientConfig()
	require.NoError(t, err)
	rootComputeDynamicKubeClient, err := dynamic.NewForConfig(rootComputeKubeconfig)
	require.NoError(t, err)

	var clusterRoleUnst *unstructured.Unstructured
	err = LoadFile("testdata/clusterrole.additional.yaml", embedded, &clusterRoleUnst)
	require.NoError(t, err)
	t.Log("Create additional clusterrole in root:compute workspace")
	_, err = rootComputeDynamicKubeClient.Resource(clusterroleGVR).Create(context.Background(), clusterRoleUnst, v1.CreateOptions{})
	if !apierrors.IsAlreadyExists(err) {
		require.NoError(t, err)
	}

	var clusterRoleBindingUnst *unstructured.Unstructured
	err = LoadFile("testdata/clusterrolebinding.additional.yaml", embedded, &clusterRoleBindingUnst)
	require.NoError(t, err)
	t.Log("Create additional clusterrolebinding in root:compute workspace")
	_, err = rootComputeDynamicKubeClient.Resource(clusterrolebindingGVR).Create(context.Background(), clusterRoleBindingUnst, v1.CreateOptions{})
	if !apierrors.IsAlreadyExists(err) {
		require.NoError(t, err)
	}
}

type reader interface {
	ReadFile(string) ([]byte, error)
}

type osReader struct {
}

func (o *osReader) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func LoadFile(path string, embedded reader, v interface{}) error {
	data, err := embedded.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, v)
}

// RunKcpEdgeCliPlugin runs the kcp workspace plugin with the provided subcommand and
// returns the combined stderr and stdout output.
func RunKcpEdgeCliPlugin(t *testing.T, kubeconfigPath string, subcommand []string) []byte {
	t.Helper()

	ctx, cancelFunc := context.WithCancel(context.Background())
	t.Cleanup(cancelFunc)

	cmdPath := filepath.Join(repositoryDir(), "cmd", "kubectl-kubestellar-syncer_gen")
	kcpCliPluginCommand := []string{"go", "run", cmdPath}

	cmdParts := append(kcpCliPluginCommand, subcommand...)
	cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)

	cmd.Env = os.Environ()
	// TODO(marun) Consider configuring the workspace plugin with args instead of this env
	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath))

	t.Logf("running: KUBECONFIG=%s %s", kubeconfigPath, strings.Join(cmdParts, " "))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Logf("kcp plugin stdout:\n%s", stdout.String())
		t.Logf("kcp plugin stderr:\n%s", stderr.String())
	}
	require.NoError(t, err, "error running kcp plugin command")
	return stdout.Bytes()
}
