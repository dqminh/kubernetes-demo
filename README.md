# Kubernetes tests

This is based on kubernetes-vagrant-coreos-cluster
```
vagrant up
./setup install
# setup proper env variables in your shell
```

This will setup:
- 1 master node
- 2 minion nodes that pods can be scheduled on
- 1 balancer node that can be used to access external service. This is so that
  we have consistent public IPs that's different from the minions' public IPs.
  This mirrors what we are probably going to have when deploy k8s to prod.

To deploy the pods:

```
kubectl create -f cfexamples/redis-controller.json
kubectl create -f cfexamples/redis-service.json
kubectl create -f cfexamples/guestbook-controller.json
kubectl create -f cfexamples/guestbook-service.json
kubectl create -f cfexamples/nginx-controller.json
kubectl create -f cfexamples/nginx-service.json
```

This setups:
- 1 redis pod
- 3 go pods
- 3 nginx pods that has the balancer public IP

I've written some tests as part of the evaluation process in addition to
architecture analysis.

- some tests in `cluster_test.go` to make sure that the cluster works
when you stop 1/multiple pods

- `cmd/test-balancer` to verify that it round-robins traffic to different
containers properly.

- Some load tests on performance of the proxy at multiple layers.

```
# access the balancer from the host ( host -> vbox -> proxy -> nginx -> proxy -> go
# echo "GET http://172.17.8.102/env" | vegeta attack -duration=10s
Duration  [total, attack, wait]    9.98487159s, 9.979999948s, 4.871642ms
Latencies [mean, 50, 95, 99, max]  7.351891ms, 7.175346ms, 13.60921ms, 28.812801ms, 28.812801ms

# access service from the balancer ( proxy -> nginx -> proxy -> go )
# echo "GET http://10.100.163.223/env" | vegeta attack -duration=10s
Duration  [total, attack, wait]    9.98175826s, 9.979033984s, 2.724276ms
Latencies [mean, 50, 95, 99, max]  4.647811ms, 4.338643ms, 10.491438ms, 18.892568ms, 18.892568ms

# access nginx container ( nginx -> proxy -> go )
# echo "GET http://10.244.99.5:8080/env" | vegeta attack -duration=10s
Duration  [total, attack, wait]    9.985572165s, 9.981208245s, 4.36392ms
Latencies [mean, 50, 95, 99, max]  3.809716ms, 3.566446ms, 9.470779ms, 29.575838ms, 29.575838ms

# access go app directly on 1 container ( go )
# echo "GET http://10.244.99.3:3000/env" | vegeta attack -duration=10s
Duration  [total, attack, wait]    9.981228268s, 9.979355936s, 1.872332ms
Latencies [mean, 50, 95, 99, max]  1.019448ms, 714.256µs, 3.159894ms, 14.446097ms, 14.446097ms
```

# kubernetes-vagrant-coreos-cluster
Turnkey **[Kubernetes](https://github.com/GoogleCloudPlatform/kubernetes)**
cluster setup with **[Vagrant](https://www.vagrantup.com)** (1.7.2+) and
**[CoreOS](https://coreos.com)**.

####If you're lazy, or in a hurry, jump to the [TL;DR](#tldr) section.

## Pre-requisites

 * **[Vagrant](https://www.vagrantup.com)**
 * a supported Vagrant hypervisor
 	* **[Virtualbox](https://www.virtualbox.org)** (the default)
 	* **[Parallels Desktop](http://www.parallels.com/eu/products/desktop/)**
 	* **[VMware Fusion](http://www.vmware.com/products/fusion)** or **[VMware Workstation](http://www.vmware.com/products/workstation)**
 * some needed userland
 	* **kubectl** (required to manage your kubernetes cluster)
 	* **fleetctl** (optional for *debugging* **[fleet](http://github.com/coreos/fleet)**)

### fleetctl

On **MacOS X** (and assuming you have [homebrew](http://brew.sh) already installed) run

```
brew update
brew install wget fleetctl
```

## Deploy Kubernetes

Current ```Vagrantfile``` will bootstrap one VM with everything needed to become a Kubernetes _master_ and, by default, a couple VMs with everything needed to become Kubernetes minions. You can however change the number of minions by setting the **NUM_INSTANCES** environment variable (explained below).
```
vagrant up
```

Verify if cluster is up & running
```
fleetctl --endpoint=http://172.17.8.101:4001 list-machines
```

NOTE: Once the installation process is complete, you should not have to provide the `--endpoint` argument.

You should see something like
```
MACHINE		IP		METADATA
dd0ee115...	172.17.8.101	role=master
74a8dc8c...	172.17.8.102	role=minion
c93da9ff...	172.17.8.103    role=minion
```

Kubernetes is ready. Now we now need to perform a few more steps, such as
* Install `kubectl` binary into */usr/local/bin* - this is needed for interacting with Kubernetes
* Set some environment variables
* Set-up Kubernetes DNS integration

Just run
```
./setup install
source ~/.bash_profile
```

You may specify a different *kubectl* version via the `KUBERNETES_VERSION` environment variable (see [here](#customization) for details).

## Clean-up

```
./setup uninstall
vagrant destroy
```

If you've set `NUM_INSTANCES` or any other variable when deploying, please make sure you set it in `vagrant destroy` call above.

## Notes about hypervisors

### Virtualbox

**VirtualBox** is the default hypervisor, and you'll probably need to disable its DHCP server
```
VBoxManage dhcpserver remove --netname HostInterfaceNetworking-vboxnet0
```

### Parallels

If you are using **Parallels Desktop**, you need to install **[vagrant-parallels](http://parallels.github.io/vagrant-parallels/docs/)** provider 
```
vagrant plugin install vagrant-parallels
```
Then just add ```--provider parallels``` to the ```vagrant up``` invocations above.

### VMware
If you are using one of the **VMware** hypervisors you must **[buy](http://www.vagrantup.com/vmware)** the matching  provider and, depending on your case, just add either ```--provider vmware-fusion``` or ```--provider vmware-workstation``` to the ```vagrant up``` invocations above.

## Private Repositories

See **DOCKERCFG** bellow.

## Customization
### Environment variables
Most aspects of your cluster setup can be customized with environment variables. Right now the available ones are:

 - **NUM_INSTANCES** sets the number of nodes (minions).

   Defaults to **2**.
 - **UPDATE_CHANNEL** sets the default CoreOS channel to be used in the VMs.

   Defaults to **alpha**.

   While by convenience, we allow an user to optionally consume CoreOS' *beta* or *stable* channels please do note that as both Kubernetes and CoreOS are quickly evolving platforms we only expect our setup to behave reliably on top of CoreOS' _alpha_ channel.
   So, **before submitting a bug**, either in [this](https://github.com/pires/kubernetes-vagrant-coreos-cluster/issues) project, or in ([Kubernetes](https://github.com/GoogleCloudPlatform/kubernetes/issues) or [CoreOS](https://github.com/coreos/bugs/issues)) **make sure it** (also) **happens in the** (default) **_alpha_ channel** :smile:
 - **COREOS_VERSION** will set the specific CoreOS release (from the given channel) to be used.

   Default is to use whatever is the **latest** one from the given channel.
 - **SERIAL_LOGGING** if set to *true* will allow logging from the VMs' serial console.

   Defaults to **false**. Only use this if you *really* know what you are doing.
 - **MASTER_MEM** sets the master's VM memory.

   Defaults to **512** (in MB)
 - **MASTER_CPUS** sets the number os vCPUs to be used by the master's VM.

   Defaults to **1**.
 - **NODE_MEM** sets the worker nodes' (aka minions in Kubernetes lingo) VM memory.

   Defaults to **1024** (in MB)
 - **NODE_CPUS** sets the number os vCPUs to be used by the minions's VMs.

   Defaults to **1**.
 - **DOCKERCFG** sets the location of your private docker repositories (and keys) configuration.

   Defaults to "**~/.dockercfg**".

   You can create/update a *~/.dockercfg* file at any time
   by running `docker login <registry>.<domain>`. All nodes will get it automatically,
   at 'vagrant up', given any modification or update to that file.

 - **KUBERNETES_VERSION** defines the specific kubernetes version being used.

   Defaults to `0.14.2`.
   Versions prior to `0.14.2` **won't work** with current cloud-config files.

 - **CLOUD_PROVIDER** defines the specific cloud provider being used. This is useful, for instance, if you're relying on kubernetes to set load-balancers for your services.

   [Possible values are `gce`, `gke`, `aws`, `azure`, `vagrant`, `vsphere`, `libvirt-coreos` and `juju`](https://github.com/GoogleCloudPlatform/kubernetes/blob/master/cluster/kube-env.sh#L17). Defaults to `vagrant`.


So, in order to start, say, a Kubernetes cluster with 3 minion nodes, 2GB of RAM and 2 vCPUs per node one just would do...

```
NODE_MEM=2048 NODE_CPUS=2 NUM_INSTANCES=3 vagrant up
```

**Please do note** that if you were using non default settings to startup your
cluster you *must* also use those exact settings when invoking
`vagrant {up,ssh,status,destroy}` to communicate with any of the nodes in the cluster as otherwise
things may not behave as you'd expect.

### Synced Folders
You can automatically mount in your *guest* VMs, at startup, an arbitrary
number of local folders in your host machine by populating accordingly the
`synced_folders.yaml` file in your `Vagrantfile` directory. For each folder
you which to mount the allowed syntax is...

```yaml
# the 'id' of this mount point. needs to be unique.
- name: foobar
# the host source directory to share with the guest(s).
  source: /foo
# the path to mount ${source} above on guest(s)
  destination: /bar
# the mount type. only NFS makes sense as, presently, we are not shipping
# hypervisor specific guest tools. defaults to `true`.
  nfs: true
# additional options to pass to the mount command on the guest(s)
# if not set the Vagrant NFS defaults will be used.
  mount_options: 'nolock,vers=3,udp,noatime'
# if the mount is enabled or disabled by default. default is `true`.
  disabled: false
```

## TL;DR

```
vagrant up
./setup install
source ~/.bash_profile
```

This will start one `master` and two `minion` nodes, download Kubernetes binaries start all needed services.
A Docker mirror cache will be provisioned in the `master`, to speed up container provisioning. This can take some time depending on your Internet connection speed.

Please do note that, at any time, you can change the number of `minions` by setting the `NUM_INSTANCES` value in subsequent `vagrant up` invocations.

### Usage

Congratulations! You're now ready to use your Kubernetes cluster.

If you just want to test something simple, start with [Kubernetes examples]
(https://github.com/GoogleCloudPlatform/kubernetes/blob/master/examples/).

For a more elaborate scenario [here]
(https://github.com/pires/kubernetes-elasticsearch-cluster) you'll find all
you need to get a scalable Elasticsearch cluster on top of Kubernetes in no
time.

## Licensing

This work is [open source](http://opensource.org/osd), and is licensed under the [Apache License, Version 2.0](http://opensource.org/licenses/Apache-2.0).
