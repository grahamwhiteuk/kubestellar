# Edge Syncer

## Build image
1. `ko build --local --platform=linux/$ARCH ./cmd/syncer`

## Install CLI Plugin (KubeStellar syncer-gen)
1. Run `make build` to build binaries
    ```
    make build
    ```
1. The new plugin to generate bootstrap manifests for kubestellar-syncer is available by adding the `./bin` directory
    ```
    Create service account and RBAC permissions in the workspace in kcp for Edge MC. Output a manifest to deploy a syncer in a WEC.

    Usage:
      syncer-gen <name> --syncer-image <kubestellar-syncer-image> -o <output-file> [flags]

    Examples:

            # Setup workspace for syncer to interact and then install syncer on a WEC
            kubectl kubestellar syncer-gen <name> --syncer-image <kubestellar-syncer-image> -o kubestellar-syncer.yaml
            KUBECONFIG=<a-wec-kubeconfig> kubectl apply -f kubestellar-syncer.yaml

            # Directly apply the manifest
            kubectl kubestellar syncer-gen <name> --syncer-image <kubestellar-syncer-image> -o - | KUBECONFIG=<a-wec-kubeconfig> kubectl apply -f -


    Flags:
    ...
    ```
 
## Edge Syncer feasibility verification

### Register kubestellar-syncer on a workload execution cluster (WEC) to connect a mailbox workspace specified by name
1. Created a mailbox workspace following to https://docs.kubestellar.io/main/Coding%20Milestones/PoC2023q1/mailbox-controller/
    ```
    $ kubectl get Workspace
    NAME                                                       TYPE        REGION   PHASE   URL                                                                                                       AGE
    1lkhy98o1f84q2a3-mb-861789a8-5867-402d-9fc4-06f0cc81fe1b   universal            Ready   https://192.168.10.105:6443/clusters/root:edge:1lkhy98o1f84q2a3-mb-861789a8-5867-402d-9fc4-06f0cc81fe1b   21s
    ```
1. Enter the mailbox workspace
    ```
    $ kubectl kcp ws 1lkhy98o1f84q2a3-mb-861789a8-5867-402d-9fc4-06f0cc81fe1b
    Current workspace is "root:edge:1lkhy98o1f84q2a3-mb-861789a8-5867-402d-9fc4-06f0cc81fe1b" (type root:universal).
    ```
1. Run kubestellar-syncer registration command
    ```
    $ kubectl kubestellar sync-gen wec1 --syncer-image $EMC_SYNCER_IMAGE -o /tmp/kubestellar-syncer.yaml
    Creating service account "kubestellar-syncer-wec1-1na3tqcd"
    Creating cluster role "kubestellar-syncer-wec1-1na3tqcd" to give service account "kubestellar-syncer-wec1-1na3tqcd"

    1. write and sync access to the synctarget "kubestellar-syncer-wec1-1na3tqcd"
    2. write access to apiresourceimports.

    Creating or updating cluster role binding "kubestellar-syncer-wec1-1na3tqcd" to bind service account "kubestellar-syncer-wec1-1na3tqcd" to cluster role "kubestellar-syncer-wec1-1na3tqcd".

    Wrote WEC manifest to /tmp/kubestellar-syncer.yaml for namespace "kubestellar-syncer-wec1-1na3tqcd". Use

      KUBECONFIG=<wec1-config> kubectl apply -f "/tmp/kubestellar-syncer.yaml"

    to apply it. Use

      KUBECONFIG=<wec1-config> kubectl get deployment -n "kubestellar-syncer-wec1-1na3tqcd" kubestellar-syncer-wec1-1na3tqcd

    to verify the syncer pod is running.
    ```
1. Deploy the generated bootstrap manifest (`/tmp/kubestellar-syncer.yaml`) to a workload execution cluster (WEC)
    ```
    $ KUBECONFIG=/tmp/kind-wec1/kubeconfig.yaml kubectl apply -f /tmp/kubestellar-syncer.yaml
    namespace/kubestellar-syncer-wec1-1na3tqcd created
    serviceaccount/kubestellar-syncer-wec1-1na3tqcd created
    secret/kubestellar-syncer-wec1-1na3tqcd-token created
    clusterrole.rbac.authorization.k8s.io/kubestellar-syncer-wec1-1na3tqcd created
    clusterrolebinding.rbac.authorization.k8s.io/kubestellar-syncer-wec1-1na3tqcd created
    role.rbac.authorization.k8s.io/kubestellar-syncer-dns-wec1-1na3tqcd created
    rolebinding.rbac.authorization.k8s.io/kubestellar-syncer-dns-wec1-1na3tqcd created
    secret/kubestellar-syncer-wec1-1na3tqcd created
    deployment.apps/kubestellar-syncer-wec1-1na3tqcd created
    ```
1. Edge Syncer successfully runs and interact with the mailbox workspace
    ```
    $ KUBECONFIG=/tmp/kind-wec1/kubeconfig.yaml kubectl get pod -A
    NAMESPACE                               NAME                                                     READY   STATUS    RESTARTS   AGE
    kubestellar-syncer-wec1-1na3tqcd   kubestellar-syncer-wec1-1na3tqcd-7467d4bf7f-7rqnt   1/1     Running   0          31s
    ...
    ```
1. Try downsync a namespace
    1. Configure downSyncResources in EdgeSyncConfig
        ```
        cat << EOL | kubectl apply -f -
        apiVersion: edge.kubestellar.io/v2alpha1
        kind: EdgeSyncConfig
        metadata:
          name: wec1
        spec:
          downSyncedResources:
          - kind: Namespace
            name: from-ws-to-wec
            version: v1
        EOL
        ```
    1. Create the namespace
        ```
        $ kubectl create ns from-ws-to-wec
        ```
    1. The namespace `from-ws-to-wec` is successfully downsynced
        ```
        $ KUBECONFIG=/tmp/kind-wec1/kubeconfig.yaml kubectl get ns
        NAME                                    STATUS   AGE
        default                                 Active   13m
        from-ws-to-wec                     Active   1s
        kubestellar-syncer-wec1-1na3tqcd   Active   11m
        kube-node-lease                         Active   13m
        kube-public                             Active   13m
        kube-system                             Active   13m
        local-path-storage                      Active   13m
        ```

### Deploy Kyverno and its policy from mailbox workspace to workload execution cluster (WEC) just by using manifests (generated from Kyverno helm chart) rather than using OLM.
1. Update EdgeSyncConfig with required resources for Helm install of Kyverno [yaml](./scripts/edge-sync-config-for-kyverno-helm.yaml)
1. Run Helm command
    ```
    $ helm install kyverno --set replicaCount=1 --namespace kyverno --create-namespace kyverno/kyverno
    NAME: kyverno
    LAST DEPLOYED: Wed Mar 22 20:43:22 2023
    NAMESPACE: kyverno
    STATUS: deployed
    ...
    ```
1. Now Kyverno is running on workload execution cluster (wec)
    ```
    $ KUBECONFIG=/tmp/kind-wec1/kubeconfig.yaml kubectl get pod -n kyverno
    NAME                      READY   STATUS    RESTARTS   AGE
    kyverno-9c494576b-dgpjt   1/1     Running   0          78s
    ```
1. Create a sample policy in the mailbox workspace to downsync 
    ```
    $ kubectl apply -f /tmp/kyverno/sample-policy.yaml
    policy.kyverno.io/sample-policy created
    ```
1. The policy is distributed to workload execution cluster (wec) and the generated policy report is upsynced
  1. On the workload execution cluster (wec)
      ```
      $ KUBECONFIG=/tmp/kind-wec1/kubeconfig.yaml kubectl get policy,policyreport
      NAME                              BACKGROUND   VALIDATE ACTION   READY
      policy.kyverno.io/sample-policy   true         enforce           true

      NAME                                            PASS   FAIL   WARN   ERROR   SKIP   AGE
      policyreport.wgpolicyk8s.io/pol-sample-policy   0      1      0      0       0      56s
      ```
  1. On the mailbox workspace
    ```
    $ kubectl get policy,policyreport
    NAME                              BACKGROUND   VALIDATE ACTION   READY
    policy.kyverno.io/sample-policy   true         enforce           true

    NAME                                            PASS   FAIL   WARN   ERROR   SKIP   AGE
    policyreport.wgpolicyk8s.io/pol-sample-policy   0      1      0      0       0      77s
    ```

### See policy reports generated at the workload execution cluster (WEC) via API Export on workload management workspace.
1. In the previous case, PolicyReport CRD is deployed as a CRD. In order to share the API across workspaces, we define PolicyReport API as APIBinding
1. Go to workload management workspace (`edge`)  
    ```
    $ kubectl kcp ws root:edge
    Current workspace is "root:edge".
    ```
1. Create APIResourceSchema and APIExport for PolicyReport CRD
    ```
    $ kubectl apply -f /tmp/kyverno/apischema.policyreports.yaml /tmp/kyverno/apiexport.policyreports.yaml
    apiresourceschema.apis.kcp.io/v0-0-1.policyreports.wgpolicyk8s.io created
    apiexport.apis.kcp.io/policy-report created
    ```
1. Create APIBindings in the mailbox workspace
    ```
    $ kubectl kcp ws root:edge:1lkhy98o1f84q2a3-mb-528a4f03-cb9b-4121-aa57-28c58ed19f22
    ```
    ```
    $ cat << EOL | kubectl apply -f -
    apiVersion: apis.kcp.io/v1alpha1
    kind: APIBinding
    metadata:
      name: policy-report
    spec:
      reference:
        export:
          path: root:edge
          name: policy-report
    EOL
    ```
1. Denature PolicyReport CRD in Kyverno Helm chart by replacing following field's value in CustomResourceDefinition for `policyreports` resource definition:
  1. Replace `metadata.name: policyreports.wgpolicyk8s.io` with `metadata.name: policyreports.wgpolicyk8s.io.denatured`
  1. Replace `spec.group: wgpolicyk8s.io` with `spec.group: wgpolicyk8s.io.denatured`
1. Deploy the Kyverno Helm yaml manifests
    ```
    kubectl create -f /tmp/kyverno/helm-install.denatured.yaml
    ```
1. Add denaturing/renaturing conversion rule to EdgeSyncConfig
    ```
    conversions:
    - upstream:
        group: apiextensions.k8s.io
        kind: CustomResourceDefinition
        name: policyreports.wgpolicyk8s.io.denatured
        version: v1
      downstream:
        group: apiextensions.k8s.io
        kind: CustomResourceDefinition
        name: policyreports.wgpolicyk8s.io
        version: v1
    ```
1. Now I can get policy reports across mailbox workspaces by one-shot from an API exposed in `edge` workspace.
    ```
    $ kubectl kcp ws root:edge
    Current workspace is "root:edge".
    ```
    ```
    $ kubectl --server="https://${ipaddr}:6443/services/apiexport/${clusterid}/policy-report/clusters/*/" get policyreports -A -o custom-columns='WORKSPACE_ID:.metadata.annotations.kcp\.io/cluster,NAME:.metadata.name,PASS:.summary.pass,FAIL:.summary.fail,WARN:.summary.warn,ERROR:.summary.error,SKIP:.summary.skip,AGE:.metadata.creationTimestamp'

    WORKSPACE_ID       NAME                PASS   FAIL   WARN   ERROR   SKIP   AGE
    1357g3bir07t1ah6   pol-sample-policy   0      1      0      0       0      2023-03-22T12:15:27Z
    1bz73lo0r5e6baep   pol-sample-policy   0      1      0      0       0      2023-03-22T12:15:27Z
    ```

### Deploy the denatured objects on mailbox workspace to workload execution cluster (WEC) by renaturing them automatically in kubestellar-syncer.
The previous case covers this item since the denatured PolicyReport CRD was downsynced and deployed as PolicyReport CRD renatured by Edge Syncer.
