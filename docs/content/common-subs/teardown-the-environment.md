<!--teardown-the-environment-start-->
To remove the example usage, delete the IMW and WMW and kind clusters run the following commands:

``` {.bash}
rm florin-syncer.yaml guilder-syncer.yaml || true
kubectl ws root
kind delete cluster --name florin
kind delete cluster --name guilder
```

Teardown of KubeStellar depends on which style of deployment was used.

### Teardown bare processes

The following command will stop whatever KubeStellar controllers are running.

``` {.bash}
kubestellar stop
```

Stop and uninstall KubeStellar and the space provider with the following command:

``` {.bash}
remove-kubestellar
```

### Teardown Kubernetes workload

With `kubectl` configured to manipulate the hosting cluster, the following command will remove the workload that is the space provider and KubeStellar.

``` {.bash}
helm delete kubestellar
```

<!--teardown-the-environment-end-->
