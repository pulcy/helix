# Helix: Pulcy Light Weight Kubernetes Cluster Provisioner

Helix uses SSH connections to nodes to be to bootstrap a highly available
Kubernetes cluster on them.

Helix supports nodes using different architectures.
So far, it supports `arm` and `amd64`.

It will bootstrap:

- A multi-node ETCD cluster
- A multi-master Kubernetes control-plane
- Any number of Kubernetes worker nodes

On each node, Helix will create the following systemd services:

- `cni-installer`: A service that downloads CNI plugins and installs the locally
- `hyperkube`: A service that pull the hyperkube docker image and copys the hyperkube binary to local disk.
- `kubelet`: A service that runs kubelet (using hyperkube binary)

Everything else is either creates a static pod in `etc/kubernetes/manifest` or
created using a normal Kubernetes resource.

## Usage

First create DNS `A` records for the APIServer of the Kubernetes cluster.
Ensure that the IP addresses of all nodes on the control-plane are listed
under a single name.

Make sure all IP addresses of nodes have a reverse DNS entry (find hostname from IP address).

Make sure your account has SSH access to all nodes.
The default SSH user is `pi`. To use a different username, set `--ssh-user=<the-user-name>`.

Then run:

```bash
helix init \
    -c <conf-dir> \
    --members=<comma-separated-list-of-node-names> \
    --apiserver=<dns-name-of-apiserver>
```

The `conf-dir` is a path of a local directory that is used to store the root certificates
and secrets for the cluster. If you later want to rebuild or extend the cluster,
use the same directory.

When the bootstrapping is complete, copy `/etc/kubernetes/admin.conf` from one
of the nodes of the control-plane to your local `kubeconfig`.

It will take some time before all services are completely up and available.
To inspect the current status, run:

```bash
kubectl get pods --all-namespaces
```

## Cleanup

To remove everything installed by Helix from all nodes of a cluster, run:

```bash
helix reset \
    --members=<comma-separated-list-of-node-names>
```

It may be needed to do a `reboot` on all nodes to clean left over docker containers.

## Components

Helix uses the following components to bootstrap the Kubernetes cluster:

- `ETCD`: As distributed key-value store (used by apiserver)
- `hyperkube`: As single binary for kubelet, kube-proxy, apiserver, controller-manager & scheduler.
- `flannel`: As network layer
- `CoreDNS`: As DNS server
